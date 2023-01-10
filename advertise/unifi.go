package advertise

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
	"github.com/sirupsen/logrus"
)

const unifiDefaultInterface = "wan"

type unifiLoginData struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type unifiMetaData struct {
	Rc  string `json:"rc"`
	Msg string `json:"msg"`
}

type unifiRoutingGetData struct {
	Id                   string `json:"_id"`
	Enabled              bool   `json:"enabled"`
	Name                 string `json:"name"`
	SiteId               string `json:"site_id"`
	Type                 string `json:"type"`
	StaticRouteInterface string `json:"static-route_interface"`
	StaticRouteNetwork   string `json:"static-route_network"`
	StaticRouteNexthop   string `json:"static-route_nexthop"`
	StaticRouteType      string `json:"static-route_type"`
}

type unifiRoutingPostData struct {
	Enabled              bool   `json:"enabled"`
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	GatewayType          string `json:"gateway_type"`
	StaticRouteInterface string `json:"static-route_interface"`
	StaticRouteNetwork   string `json:"static-route_network"`
	StaticRouteNexthop   string `json:"static-route_nexthop"`
	StaticRouteType      string `json:"static-route_type"`
}

type unifiRoutingRespData struct {
	Data []unifiRoutingGetData `json:"data"`
	Meta unifiMetaData         `json:"meta"`
}

type unifiRespData struct {
	Meta unifiMetaData `json:"meta"`
}

type unifiLoginRespData struct {
	UniqueId string `json:"unique_id"`
}

type unifiRoute struct {
	Id      string
	Name    string
	Network string
	Nexthop string
	Enabled bool
}

type unifiPortGetData struct {
	Id      string `json:"_id"`
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	Src     string `json:"src"`
	DstPort string `json:"dst_port"`
	Fwd     string `json:"fwd"`
	FwdPort string `json:"fwd_port"`
	Proto   string `json:"proto"`
	SiteId  string `json:"site_id"`
}

type unifiPortPostData struct {
	Enabled       bool   `json:"enabled"`
	Name          string `json:"name"`
	Src           string `json:"src"`
	DstPort       string `json:"dst_port"`
	Fwd           string `json:"fwd"`
	FwdPort       string `json:"fwd_port"`
	PfwdInterface string `json:"pfwd_interface"` // wan,wan2,both
	Proto         string `json:"proto"`
}

type unifiPortRespData struct {
	Data []unifiPortGetData `json:"data"`
	Meta unifiMetaData      `json:"meta"`
}

type unifiPortForward struct {
	Id          string
	Enabled     bool
	Name        string
	Source      string
	DestPort    string
	Forward     string
	ForwardPort string
	Proto       string
}

func site() string {
	site := config.Config.Unifi.Site
	if site == "" {
		site = "default"
	}
	return site
}

func unifiGetCsrf(client *http.Client) (token string, err error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s", config.Config.Unifi.Controller),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Csrf request error"),
		}
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi login request error"),
		}
		return
	}
	defer resp.Body.Close()

	token = resp.Header.Get("X-CSRF-Token")
	if token == "" {
		err = &errortypes.RequestError{
			errors.New("advertise: Unifi csrf token empty"),
		}
		return
	}

	return
}

func unifiGetClient() (client *http.Client, csrfToken string, err error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: CookieJar error"),
		}
		return
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client = &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   10 * time.Second,
	}

	csrfToken, err = unifiGetCsrf(client)
	if err != nil {
		return
	}

	data := &unifiLoginData{
		Username:   config.Config.Unifi.Username,
		Password:   config.Config.Unifi.Password,
		RememberMe: false,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Json parse error"),
		}
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/auth/login", config.Config.Unifi.Controller),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi login request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Failed to read Unifi response"),
		}
		return
	}

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Failed to login to Unifi"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Error("advertise: Failed to login to Unifi bad status")

		return
	}

	respData := &unifiLoginRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Failed to parse login response"),
		}
		return
	}

	if respData.UniqueId == "" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Failed to login to Unifi"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Error("advertise: Failed to login to Unifi invalid response")

		return
	}

	csrfToken = resp.Header.Get("X-CSRF-Token")

	return
}

func unifiGetRoutes(client *http.Client, csrfToken string) (
	routes []*unifiRoute, err error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/proxy/network/api/s/%s/rest/routing",
			config.Config.Unifi.Controller, site()),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi response error"),
		}
		return
	}

	respData := &unifiRoutingRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Unifi parse error"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi api error"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Unifi api error")

		return
	}

	routes = []*unifiRoute{}

	for _, routeData := range respData.Data {
		if routeData.Type != "static-route" {
			continue
		}

		route := &unifiRoute{
			Id:      routeData.Id,
			Name:    routeData.Name,
			Network: routeData.StaticRouteNetwork,
			Nexthop: routeData.StaticRouteNexthop,
			Enabled: routeData.Enabled,
		}

		routes = append(routes, route)
	}

	return
}

func unifiDeleteRoute(client *http.Client, csrfToken, id string) (err error) {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/proxy/network/api/s/%s/rest/routing/%s",
			config.Config.Unifi.Controller, site(), id),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi response error"),
		}
		return
	}

	respData := &unifiRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Unifi parse error"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi api error"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Unifi api error")

		return
	}

	return
}

func unifiAddRoute(client *http.Client, csrfToken, network, nexthop string) (
	err error) {

	data := &unifiRoutingPostData{
		Enabled: true,
		Name: fmt.Sprintf(
			"pritunl-link-%x", md5.Sum([]byte(network))),
		Type:               "static-route",
		GatewayType:        "default",
		StaticRouteNetwork: network,
		StaticRouteNexthop: nexthop,
		StaticRouteType:    "nexthop-route",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Json parse error"),
		}
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/proxy/network/api/s/%s/rest/routing",
			config.Config.Unifi.Controller, site()),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi response error"),
		}
		return
	}

	respData := &unifiRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Unifi parse error"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi api error"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Unifi api error")

		return
	}

	return
}

func unifiHasRoute(client *http.Client, csrfToken, network, nexthop string) (
	exists bool, err error) {

	rts, err := unifiGetRoutes(client, csrfToken)
	if err != nil {
		return
	}

	for _, route := range rts {
		if route.Network == network {
			if route.Enabled && route.Nexthop == nexthop {
				exists = true
				return
			}

			err = unifiDeleteRoute(client, csrfToken, route.Id)
			if err != nil {
				return
			}

			return
		}
	}

	return
}

func UnifiAddRoute(network string) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	nexthop := ""
	if strings.Contains(network, ":") {
		nexthop = state.GetAddress6()
	} else {
		nexthop = state.GetLocalAddress()
	}
	if nexthop == "" {
		logrus.WithFields(logrus.Fields{
			"nexthop":  state.GetLocalAddress(),
			"nexthop6": state.GetAddress6(),
		}).Error("advertise: Missing local address " +
			"skipping route advertisement")
		return
	}

	client, csrfToken, err := unifiGetClient()
	if err != nil {
		return
	}

	exists, err := unifiHasRoute(client, csrfToken, network, nexthop)
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddRoute(client, csrfToken, network, nexthop)
		if err != nil {
			return
		}
	}

	route := &routes.UnifiRoute{
		Network: network,
		Nexthop: nexthop,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}

func UnifiDeleteRoute(route *routes.UnifiRoute) (err error) {
	if config.Config.DeleteRoutes {
		if constants.Interrupt {
			err = &errortypes.UnknownError{
				errors.Wrap(err, "advertise: Interrupt"),
			}
			return
		}

		client, csrfToken, e := unifiGetClient()
		if e != nil {
			err = e
			return
		}

		rts, e := unifiGetRoutes(client, csrfToken)
		if e != nil {
			err = e
			return
		}

		for _, rte := range rts {
			if rte.Network == route.Network && rte.Nexthop == route.Nexthop {
				err = unifiDeleteRoute(client, csrfToken, rte.Id)
				if err != nil {
					return
				}

				break
			}
		}
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}

func unifiGetPorts(client *http.Client, csrfToken string) (
	ports []*unifiPortForward, err error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/proxy/network/api/s/%s/rest/portforward",
			config.Config.Unifi.Controller, site()),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi response error"),
		}
		return
	}

	respData := &unifiPortRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Unifi parse error"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi api error"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Unifi api error")

		return
	}

	ports = []*unifiPortForward{}

	for _, portData := range respData.Data {
		port := &unifiPortForward{
			Id:          portData.Id,
			Enabled:     portData.Enabled,
			Name:        portData.Name,
			Source:      portData.Src,
			DestPort:    portData.DstPort,
			Forward:     portData.Fwd,
			ForwardPort: portData.FwdPort,
			Proto:       portData.Proto,
		}

		ports = append(ports, port)
	}

	return
}
func unifiDeletePort(client *http.Client, csrfToken, id string) (err error) {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/proxy/network/api/s/%s/rest/portforward/%s",
			config.Config.Unifi.Controller, site(), id),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi response error"),
		}
		return
	}

	respData := &unifiRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Unifi parse error"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi api error"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Unifi api error")

		return
	}

	return
}

func unifiAddPort(client *http.Client, csrfToken, source, destPort,
	forward, forwardPort, proto string) (err error) {

	iface := config.Config.Unifi.Interface
	if iface == "" {
		iface = unifiDefaultInterface
	}
	iface = strings.ToLower(iface)

	data := &unifiPortPostData{
		Enabled:       true,
		Name:          fmt.Sprintf("pritunl-link-%s", forwardPort),
		Src:           source,
		DstPort:       destPort,
		Fwd:           forward,
		FwdPort:       forwardPort,
		PfwdInterface: iface,
		Proto:         proto,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Json parse error"),
		}
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/proxy/network/api/s/%s/rest/portforward",
			config.Config.Unifi.Controller, site()),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("X-CSRF-Token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi request error"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi response error"),
		}
		return
	}

	respData := &unifiRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Unifi parse error"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Unifi api error"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Unifi api error")

		return
	}

	return
}

func unifiHasPort(client *http.Client, csrfToken string,
	ports []*unifiPortForward, source, destPort, forward, forwardPort,
	proto string) (exists bool, err error) {

	for _, port := range ports {
		if (port.DestPort == destPort && (port.Proto == proto ||
			port.Proto == "tcp_udp")) || (port.Forward == forward &&
			port.ForwardPort == forwardPort && (port.Proto == proto ||
			port.Proto == "tcp_udp")) {

			if port.Enabled && port.Source == source &&
				port.Forward == forward && port.ForwardPort == forwardPort &&
				port.Proto == proto {

				exists = true
				return
			}

			err = unifiDeletePort(client, csrfToken, port.Id)
			if err != nil {
				return
			}

			return
		}
	}

	return
}

func UnifiAddPorts() (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	source := "any"
	forward := state.GetLocalAddress()
	proto := "udp"

	if forward == "" {
		return
	}

	client, csrfToken, err := unifiGetClient()
	if err != nil {
		return
	}

	ports, err := unifiGetPorts(client, csrfToken)
	if err != nil {
		return
	}

	exists, err := unifiHasPort(client, csrfToken, ports, source, "500",
		forward, "500", proto)
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddPort(client, csrfToken, source, "500",
			forward, "500", proto)
		if err != nil {
			return
		}
	}

	exists, err = unifiHasPort(client, csrfToken, ports, source, "4500",
		forward, "4500", proto)
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddPort(client, csrfToken, source, "4500",
			forward, "4500", proto)
		if err != nil {
			return
		}
	}

	exists, err = unifiHasPort(client, csrfToken, ports, source, "9790",
		forward, "9790", "tcp")
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddPort(client, csrfToken, source, "9790",
			forward, "9790", "tcp")
		if err != nil {
			return
		}
	}

	return
}
