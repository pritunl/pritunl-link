package advertise

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const unifiDefaultInterface = "WAN1"

type unifiLoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Strict   bool   `json:"strict"`
	Remember bool   `json:"remember"`
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

type unifiRoute struct {
	Id      string
	Name    string
	Network string
	Nexthop string
	Enabled bool
}

type unifiPortGetData struct {
	Id      string `json:"_id"`
	Name    string `json:"name"`
	Src     string `json:"src"`
	DstPort string `json:"dst_port"`
	Fwd     string `json:"fwd"`
	FwdPort string `json:"fwd_port"`
	Proto   string `json:"proto"`
	SiteId  string `json:"site_id"`
}

type unifiPortPostData struct {
	Name    string `json:"name"`
	Src     string `json:"src"`
	DstPort string `json:"dst_port"`
	Fwd     string `json:"fwd"`
	FwdPort string `json:"fwd_port"`
	Proto   string `json:"proto"`
}

type unifiPortRespData struct {
	Data []unifiPortGetData `json:"data"`
	Meta unifiMetaData      `json:"meta"`
}

type unifiPortForward struct {
	Id          string
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

func unifiGetClient() (client *http.Client, err error) {
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
		Timeout:   60 * time.Second,
	}

	data := &unifiLoginData{
		Username: config.Config.Unifi.Username,
		Password: config.Config.Unifi.Password,
		Strict:   false,
		Remember: false,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Json parse error"),
		}
		return
	}

	routeInterface := config.Config.Unifi.Interface
	if routeInterface == "" {
		routeInterface = unifiDefaultInterface
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/login", config.Config.Unifi.Controller),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")

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

	respData := &unifiRespData{}

	err = json.Unmarshal(body, respData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "advertise: Failed to parse login response"),
		}
		return
	}

	if respData.Meta.Rc != "ok" {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Failed to login to Unifi"),
		}

		logrus.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(body),
			"error":    err,
		}).Info("advertise: Failed to login to Unifi")

		return
	}

	return
}

func unifiGetRoutes(client *http.Client) (routes []*unifiRoute, err error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/s/%s/rest/routing",
			config.Config.Unifi.Controller, site()),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

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

func unifiDeleteRoute(client *http.Client, id string) (err error) {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/api/s/%s/rest/routing/%s",
			config.Config.Unifi.Controller, site(), id),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

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

func unifiAddRoute(client *http.Client, network, nexthop string) (err error) {
	data := &unifiRoutingPostData{
		Enabled: true,
		Name: fmt.Sprintf(
			"pritunl-%x", md5.Sum([]byte(network))),
		Type:                 "static-route",
		StaticRouteInterface: "WAN1",
		StaticRouteNetwork:   network,
		StaticRouteNexthop:   nexthop,
		StaticRouteType:      "nexthop-route",
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
		fmt.Sprintf("%s/api/s/%s/rest/routing",
			config.Config.Unifi.Controller, site()),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

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

func unifiHasRoute(client *http.Client, network, nexthop string) (
	exists bool, err error) {

	rts, err := unifiGetRoutes(client)
	if err != nil {
		return
	}

	for _, route := range rts {
		if route.Network == network {
			if route.Enabled && route.Nexthop == nexthop {
				exists = true
				return
			}

			err = unifiDeleteRoute(client, route.Id)
			if err != nil {
				return
			}

			return
		}
	}

	return
}

func UnifiAddRoute(network string) (err error) {
	nexthop := state.GetLocalAddress()

	client, err := unifiGetClient()
	if err != nil {
		return
	}

	exists, err := unifiHasRoute(client, network, nexthop)
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddRoute(client, network, nexthop)
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
		client, e := unifiGetClient()
		if e != nil {
			err = e
			return
		}

		rts, e := unifiGetRoutes(client)
		if e != nil {
			err = e
			return
		}

		for _, rte := range rts {
			if rte.Network == route.Network && rte.Nexthop == route.Nexthop {
				err = unifiDeleteRoute(client, rte.Id)
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

func unifiGetPorts(client *http.Client) (
	ports []*unifiPortForward, err error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/s/%s/rest/portforward",
			config.Config.Unifi.Controller, site()),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

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
func unifiDeletePort(client *http.Client, id string) (err error) {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/api/s/%s/rest/portforward/%s",
			config.Config.Unifi.Controller, site(), id),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

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

func unifiAddPort(client *http.Client, source, destPort,
	forward, forwardPort, proto string) (err error) {

	data := &unifiPortPostData{
		Name:    "Pritunl IPsec",
		Src:     source,
		DstPort: destPort,
		Fwd:     forward,
		FwdPort: forwardPort,
		Proto:   proto,
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
		fmt.Sprintf("%s/api/s/%s/rest/portforward",
			config.Config.Unifi.Controller, site()),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "advertise: Request error"),
		}
		return
	}

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

func unifiHasPort(client *http.Client, ports []*unifiPortForward, source,
	destPort, forward, forwardPort, proto string) (exists bool, err error) {

	for _, port := range ports {
		if (port.DestPort == destPort && (port.Proto == proto ||
			port.Proto == "tcp_udp")) || (port.Forward == forward &&
			port.ForwardPort == forwardPort && (port.Proto == proto ||
			port.Proto == "tcp_udp")) {

			if port.Source == source && port.Forward == forward &&
				port.ForwardPort == forwardPort && port.Proto == proto {

				exists = true
				return
			}

			err = unifiDeletePort(client, port.Id)
			if err != nil {
				return
			}

			return
		}
	}

	return
}

func UnifiAddPorts() (err error) {
	source := "any"
	forward := state.GetLocalAddress()
	proto := "udp"

	client, err := unifiGetClient()
	if err != nil {
		return
	}

	ports, err := unifiGetPorts(client)
	if err != nil {
		return
	}

	exists, err := unifiHasPort(client, ports, source, "500",
		forward, "500", proto)
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddPort(client, source, "500",
			forward, "500", proto)
		if err != nil {
			return
		}
	}

	exists, err = unifiHasPort(client, ports, source, "4500",
		forward, "4500", proto)
	if err != nil {
		return
	}

	if !exists {
		err = unifiAddPort(client, source, "4500",
			forward, "4500", proto)
		if err != nil {
			return
		}
	}

	return
}
