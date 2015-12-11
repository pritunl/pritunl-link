package profile

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/utils"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func AuthReq(token, secret, baseUrl, method, path string, data interface{}) (
	resp *http.Response, err error) {

	method = strings.ToUpper(method)

	trans := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: trans,
	}

	var body io.Reader
	var encData []byte
	if data != nil {
		encData, err = json.Marshal(data)
		if err != nil {
			err = &errortypes.ParseError{
				errors.Wrap(err, "profile: Failed to parse data"),
			}
			return
		}
		body = bytes.NewBuffer(encData)
	}

	req, err := http.NewRequest(method, baseUrl+path, body)
	if err != nil {
		err = errortypes.RequestError{
			errors.Wrap(err, "profile: Unknown request parse error"),
		}
		return
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := utils.Uuid()
	auth := []string{
		token,
		timestamp,
		nonce,
		method,
		path,
	}
	if encData != nil {
		auth = append(auth, string(encData))
	}
	authStr := strings.Join(auth, "&")

	hashFunc := hmac.New(sha256.New, []byte(secret))
	hashFunc.Write([]byte(authStr))
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	req.Header.Set("Auth-Token", token)
	req.Header.Set("Auth-Timestamp", timestamp)
	req.Header.Set("Auth-Nonce", nonce)
	req.Header.Set("Auth-Signature", sig)

	if encData != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err = client.Do(req)
	if err != nil {
		err = errortypes.RequestError{
			errors.Wrap(err, "profile: Unknown request error"),
		}
		return
	}

	return
}
