package advertise

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
)

var pritunlClient = &http.Client{
	Timeout: 6 * time.Second,
}

type pritunlRoute struct {
	Destination string `bson:"destination" json:"destination"`
	Target      string `bson:"target" json:"target"`
	Link        bool   `bson:"link" json:"link"`
}

func pritunlGetRoutes(hostname, orgId, vpcId, token, secret string) (
	vpcRoutes []*pritunlRoute, err error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://%s/vpc/%s/routes", hostname, vpcId),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Request init error"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if orgId != "" {
		req.Header.Set("Organization", orgId)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	nonce, err := utils.RandStr(32)
	if err != nil {
		return
	}

	authStr := strings.Join([]string{
		token,
		timestamp,
		nonce,
		"GET",
		fmt.Sprintf("/vpc/%s/routes", vpcId),
	}, "&")

	hashFunc := hmac.New(sha512.New, []byte(secret))
	hashFunc.Write([]byte(authStr))
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	req.Header.Set("Auth-Token", token)
	req.Header.Set("Auth-Timestamp", timestamp)
	req.Header.Set("Auth-Nonce", nonce)
	req.Header.Set("Auth-Signature", sig)

	res, err := pritunlClient.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Request put error"),
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "cloud: Bad status %n code from server",
				res.StatusCode),
		}
		return
	}

	vpcRoutes = []*pritunlRoute{}
	err = json.NewDecoder(res.Body).Decode(&vpcRoutes)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "cloud: Failed to parse response body"),
		}
		return
	}

	return
}

func pritunlUpdateRoutes(hostname, orgId, vpcId, token, secret string,
	vpcRoutes []*pritunlRoute) (err error) {

	dataBuf := &bytes.Buffer{}
	err = json.NewEncoder(dataBuf).Encode(vpcRoutes)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "cloud: Failed to marshal routes"),
		}
		return
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://%s/vpc/%s/routes", hostname, vpcId),
		dataBuf,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Request init error"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if orgId != "" {
		req.Header.Set("Organization", orgId)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	nonce, err := utils.RandStr(32)
	if err != nil {
		return
	}

	authStr := strings.Join([]string{
		token,
		timestamp,
		nonce,
		"PUT",
		fmt.Sprintf("/vpc/%s/routes", vpcId),
	}, "&")

	hashFunc := hmac.New(sha512.New, []byte(secret))
	hashFunc.Write([]byte(authStr))
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	req.Header.Set("Auth-Token", token)
	req.Header.Set("Auth-Timestamp", timestamp)
	req.Header.Set("Auth-Nonce", nonce)
	req.Header.Set("Auth-Signature", sig)

	res, err := pritunlClient.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Request put error"),
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "cloud: Bad status %n code from server",
				res.StatusCode),
		}
		return
	}

	return
}

func PritunlAddRoute(network string) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	target := ""
	if strings.Contains(network, ":") {
		target = state.GetAddress6()
	} else {
		target = state.GetLocalAddress()
	}
	if target == "" {
		logrus.WithFields(logrus.Fields{
			"target":  state.GetLocalAddress(),
			"target6": state.GetAddress6(),
		}).Error("advertise: Missing local address " +
			"skipping route advertisement")
		return
	}

	hostname := config.Config.Pritunl.Hostname
	orgId := config.Config.Pritunl.OrganizationId
	vpcId := config.Config.Pritunl.VpcId
	token := config.Config.Pritunl.Token
	secret := config.Config.Pritunl.Secret

	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	vpcRoutes, err := pritunlGetRoutes(
		hostname, orgId, vpcId, token, secret)
	if err != nil {
		return
	}

	exists := false
	updated := false

	for _, route := range vpcRoutes {
		if route.Destination == network {
			exists = true
			if route.Target != target {
				route.Target = target
				updated = true
			}
		}
	}

	if !exists {
		vpcRoutes = append(vpcRoutes, &pritunlRoute{
			Destination: network,
			Target:      target,
		})
		updated = true
	}

	if updated {
		err = pritunlUpdateRoutes(
			hostname, orgId, vpcId, token, secret, vpcRoutes)
		if err != nil {
			return
		}
	}

	route := &routes.PritunlRoute{
		DestNetwork:    network,
		OrganizationId: orgId,
		VpcId:          vpcId,
		Target:         target,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}

func PritunlDeleteRoute(route *routes.PritunlRoute) (err error) {
	if config.Config.DeleteRoutes {
		if constants.Interrupt {
			err = &errortypes.UnknownError{
				errors.Wrap(err, "advertise: Interrupt"),
			}
			return
		}

		hostname := config.Config.Pritunl.Hostname
		token := config.Config.Pritunl.Token
		secret := config.Config.Pritunl.Secret

		vpcRoutes, e := pritunlGetRoutes(hostname,
			route.OrganizationId, route.VpcId, token, secret)
		if e != nil {
			err = e
			return
		}

		updated := false

		for i, rte := range vpcRoutes {
			if rte.Destination == route.DestNetwork &&
				rte.Target == route.Target {

				updated = true
				vpcRoutes = append(vpcRoutes[:i], vpcRoutes[i+1:]...)
			}
		}

		if updated {
			err = pritunlUpdateRoutes(hostname, route.OrganizationId,
				route.VpcId, token, secret, vpcRoutes)
			if err != nil {
				return
			}
		}
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}
