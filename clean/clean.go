package clean

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/iptables"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
)

func cleanup(uri string) (err error) {
	uriData, err := url.ParseRequestURI(uri)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "clean: Failed to parse uri"),
		}
		return
	}

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("https://%s/link/state", uriData.Host),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "clean: Request init error"),
		}
		return
	}

	hostId := uriData.User.Username()
	hostSecret, _ := uriData.User.Password()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	nonce, err := utils.RandStr(32)
	if err != nil {
		return
	}

	authStr := strings.Join([]string{
		hostId,
		timestamp,
		nonce,
		"DELETE",
		"/link/state",
	}, "&")

	hashFunc := hmac.New(sha512.New, []byte(hostSecret))
	hashFunc.Write([]byte(authStr))
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	req.Header.Set("Auth-Token", hostId)
	req.Header.Set("Auth-Timestamp", timestamp)
	req.Header.Set("Auth-Nonce", nonce)
	req.Header.Set("Auth-Signature", sig)

	var client *http.Client
	if config.Config.SkipVerify {
		client = state.ClientInsec
	} else {
		client = state.ClientSec
	}

	res, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "clean: Request delete error"),
		}
		return
	}
	defer res.Body.Close()

	return
}

func CleanUp() {
	uris := config.Config.Uris

	iptables.ClearIpTables()
	ipsec.DelDirectRoute()
	ipsec.StopTunnel()

	for _, uri := range uris {
		go cleanup(uri)
	}

	time.Sleep(3 * time.Second)

	return
}
