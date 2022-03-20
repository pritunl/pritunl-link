package advertise

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type edgeRouteNextHop struct {
	Type      string `json:"t"`
	Metric    string `json:"metric"`
	Interface string `json:"intf"`
	Via       string `json:"via"`
}

type edgeRoutesOutput struct {
	Prefix  string             `json:"pfx"`
	NextHop []edgeRouteNextHop `json:"nh"`
}

type edgeRoutesData struct {
	Output []edgeRoutesOutput `json:"output"`
}

type edgeRoute struct {
	Destination string
	NextHop     string
	Interface   string
}

type edgeFeatureRule struct {
	Description      string `json:"description"`
	ForwardToAddress string `json:"forward-to-address"`
	ForwardToPort    string `json:"forward-to-port"`
	OriginalPort     string `json:"original-port"`
	Protocol         string `json:"protocol"`
}

type edgeFeatureData struct {
	AutoFirewall string          `json:"auto-firewall"`
	HairpinNat   string          `json:"hairpin-nat"`
	LansConfig   json.RawMessage `json:"lans-config"`
	RulesConfig  json.RawMessage `json:"rules-config"`
	Wan          string          `json:"wan"`
}

type edgeFeature struct {
	Data    edgeFeatureData `json:"data"`
	Success string          `json:"success"`
	Error   string          `json:"error"`
}

type edgeFeatureResp struct {
	Feature edgeFeature `json:"FEATURE"`
	Success bool        `json:"success"`
}

type edgeFeatureApplyData struct {
	Action   string          `json:"action"`
	Apply    edgeFeatureData `json:"apply"`
	Scenario string          `json:"scenario"`
}

type edgeFeatureApply struct {
	Data edgeFeatureApplyData `json:"data"`
}

type edgeFeatureApplyStatus struct {
	Success string `json:"success"`
	Error   string `json:"error"`
}

type edgeFeatureApplyResp struct {
	Feature edgeFeatureApplyStatus `json:"FEATURE"`
	Success bool                   `json:"success"`
}

func edgeGetClient() (client *http.Client, err error) {
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Failed to create cookie jar"),
		}
		return
	}

	client = &http.Client{
		Timeout: 10 * time.Second,
		Jar:     cookieJar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	loginData := url.Values{
		"username": []string{config.Config.Edge.Username},
		"password": []string{config.Config.Edge.Password},
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s", config.Config.Edge.Hostname),
		strings.NewReader(loginData.Encode()),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge login request error"),
		}
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge login request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf(
				"edge: Edge login bad status %d",
				resp.StatusCode,
			),
		}
		return
	}

	return
}

func edgeGetCsrfToken(client *http.Client) (token string, err error) {
	edgeUrl, err := url.Parse(fmt.Sprintf("https://%s",
		config.Config.Edge.Hostname))
	if err != nil {
		return
	}

	cookies := client.Jar.Cookies(edgeUrl)

	for _, cookie := range cookies {
		cookieStr := cookie.String()

		if strings.HasPrefix(cookieStr, "X-CSRF-TOKEN=") {
			cookieSpl := strings.Split(cookieStr, "X-CSRF-TOKEN=")
			if len(cookieSpl) != 2 {
				err = &errortypes.ParseError{
					errors.New("edge: Edge cookie len invalid"),
				}
				return
			}

			token = cookieSpl[1]
			return
		}
	}

	if token == "" {
		err = &errortypes.ParseError{
			errors.New("edge: Edge cookie csrf token null"),
		}
		return
	}

	return
}

func edgeGetRoutes(client *http.Client) (routes []*edgeRoute, err error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://%s/api/edge/data.json?data=routes",
			config.Config.Edge.Hostname),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge routes request error"),
		}
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge routes request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf(
				"edge: Edge routes bad status %d",
				resp.StatusCode,
			),
		}
		return
	}

	data := &edgeRoutesData{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Failed to parse edge routes data"),
		}
		return
	}

	if data.Output == nil || len(data.Output) == 0 {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Edge routes nil output"),
		}
		return
	}

	routes = []*edgeRoute{}

	for _, output := range data.Output {
		destination := output.Prefix

		if output.NextHop == nil || len(output.NextHop) == 0 {
			continue
		}

		for _, nextHop := range output.NextHop {
			route := &edgeRoute{
				Destination: destination,
				NextHop:     nextHop.Via,
				Interface:   nextHop.Interface,
			}
			routes = append(routes, route)
		}
	}

	return
}

func edgeAddRoute(destination string) (err error) {
	nexthop := state.GetLocalAddress()

	client, err := edgeGetClient()
	if err != nil {
		return
	}

	rtes, err := edgeGetRoutes(client)
	if err != nil {
		return
	}

	for _, route := range rtes {
		if route.Destination == destination && route.NextHop == nexthop {
			return
		}
	}

	csrfToken, err := edgeGetCsrfToken(client)
	if err != nil {
		return
	}

	data := map[string]interface{}{
		"DELETE": map[string]interface{}{
			"protocols": map[string]interface{}{
				"static": map[string]interface{}{
					"route": map[string]interface{}{
						destination: nil,
					},
				},
			},
		},
		"SET": map[string]interface{}{
			"protocols": map[string]interface{}{
				"static": map[string]interface{}{
					"route": map[string]interface{}{
						destination: map[string]interface{}{
							"next-hop": map[string]interface{}{
								nexthop: map[string]interface{}{
									"description": "pritunl-zero",
								},
							},
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Json parse error"),
		}
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s/api/edge/batch.json",
			config.Config.Edge.Hostname),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge set batch request error"),
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge set batch request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf(
				"edge: Edge set batch bad status %d",
				resp.StatusCode,
			),
		}
		return
	}

	return
}

func edgeDeleteRoute(route *routes.EdgeRoute) (err error) {
	if !config.Config.DeleteRoutes {
		err = route.Remove()
		if err != nil {
			return
		}

		return
	}

	client, err := edgeGetClient()
	if err != nil {
		return
	}

	csrfToken, err := edgeGetCsrfToken(client)
	if err != nil {
		return
	}

	data := map[string]interface{}{
		"SET": map[string]interface{}{
			"protocols": map[string]interface{}{
				"static": map[string]interface{}{
					"route": map[string]interface{}{
						route.Nexthop: nil,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Json parse error"),
		}
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s/api/edge/batch.json",
			config.Config.Edge.Hostname),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge delete batch request error"),
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge delete batch request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf(
				"edge: Edge delete batch bad status %d",
				resp.StatusCode,
			),
		}
		return
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}

func edgeGetRules(client *http.Client) (rules *edgeFeatureResp, err error) {
	csrfToken, err := edgeGetCsrfToken(client)
	if err != nil {
		return
	}

	getData := map[string]interface{}{
		"data": map[string]string{
			"action":   "load",
			"scenario": ".Port_Forwarding",
		},
	}

	jsonData, err := json.Marshal(getData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Json parse error"),
		}
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s/api/edge/feature.json",
			config.Config.Edge.Hostname),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge feature load request error"),
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge feature load request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf(
				"edge: Edge feature load bad status %d",
				resp.StatusCode,
			),
		}
		return
	}

	rules = &edgeFeatureResp{}
	err = json.NewDecoder(resp.Body).Decode(rules)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Failed to parse feature load data"),
		}
		return
	}

	if !rules.Success || rules.Feature.Success == "0" {
		err = &errortypes.ParseError{
			errors.Wrapf(err, "edge: Feature load error '%s'",
				rules.Feature.Error),
		}
		return
	}
	return
}

func edgeAddPorts() (err error) {
	nexthop := state.GetLocalAddress()

	client, err := edgeGetClient()
	if err != nil {
		return
	}

	rules, err := edgeGetRules(client)
	if err != nil {
		return
	}

	port500 := false
	port4500 := false
	oldRules := []edgeFeatureRule{}
	newRules := []edgeFeatureRule{}

	_ = json.Unmarshal(rules.Feature.Data.RulesConfig, &oldRules)

	for _, rule := range oldRules {
		if rule.OriginalPort == "500" {
			if rule.Protocol == "udp" &&
				rule.ForwardToPort == "500" &&
				rule.Description == "pritunl-zero" &&
				rule.ForwardToAddress == nexthop {

				port500 = true
			} else {
				continue
			}
		} else if rule.OriginalPort == "4500" {
			if rule.Protocol == "udp" &&
				rule.ForwardToPort == "4500" &&
				rule.Description == "pritunl-zero" &&
				rule.ForwardToAddress == nexthop {

				port4500 = true
			} else {
				continue
			}
		}
	}

	if port500 && port4500 {
		return
	}

	if !port500 {
		rule := edgeFeatureRule{
			Description:      "pritunl-zero",
			ForwardToAddress: nexthop,
			ForwardToPort:    "500",
			OriginalPort:     "500",
			Protocol:         "udp",
		}

		newRules = append(newRules, rule)
	}
	if !port4500 {
		rule := edgeFeatureRule{
			Description:      "pritunl-zero",
			ForwardToAddress: nexthop,
			ForwardToPort:    "4500",
			OriginalPort:     "4500",
			Protocol:         "udp",
		}

		newRules = append(newRules, rule)
	}

	rulesByt, err := json.Marshal(newRules)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Failed to marshal rules"),
		}
		return
	}
	rules.Feature.Data.RulesConfig = rulesByt

	apply := &edgeFeatureApply{
		Data: edgeFeatureApplyData{
			Action:   "apply",
			Apply:    rules.Feature.Data,
			Scenario: ".Port_Forwarding",
		},
	}

	applyData, err := json.Marshal(apply)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Json parse error"),
		}
		return
	}

	csrfToken, err := edgeGetCsrfToken(client)
	if err != nil {
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s/api/edge/feature.json",
			config.Config.Edge.Hostname),
		bytes.NewBuffer(applyData),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge feature apply request error"),
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "edge: Edge feature apply request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf(
				"edge: Edge feature apply bad status %d",
				resp.StatusCode,
			),
		}
		return
	}

	data := &edgeFeatureApplyResp{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "edge: Failed to parse feature apply data"),
		}
		return
	}

	if !data.Success || data.Feature.Success == "0" {
		err = &errortypes.ParseError{
			errors.Wrapf(err, "edge: Feature apply error '%s'",
				data.Feature.Error),
		}
		return
	}

	return
}

func EdgeAddRoute(destination string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%s", r))
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("edge: Edge add route recover")
			return
		}
	}()

	err = edgeAddRoute(destination)
	return
}

func EdgeDeleteRoute(route *routes.EdgeRoute) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%s", r))
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("edge: Edge delete route recover")
			return
		}
	}()

	err = edgeDeleteRoute(route)
	return
}

func EdgeAddPorts() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%s", r))
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("edge: Edge add ports recover")
			return
		}
	}()

	err = edgeAddPorts()
	return
}
