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
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type authUserData struct {
	Username     string   `json:"username"`
	NetworkLinks []string `json:"network_links"`
}

func getPath() string {
	return filepath.Join(ConfDir, Username) + ".json"
}

func AuthReq(token, secret string, hash func() hash.Hash, baseUrl, method,
	path string, data interface{}) (

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

	hashFunc := hmac.New(hash, []byte(secret))
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

func GetProfiles() (prfls []*Profile, err error) {
	data := authUserData{
		Username:     Username,
		NetworkLinks: NetworkLinks,
	}

	resp, err := AuthReq(Token, Secret, sha256.New, Host,
		"POST", "/auth/user", data)
	if err != nil {
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = errortypes.RequestError{
			errors.Newf("profile: Bad status code %d from auth",
				resp.StatusCode),
		}
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	prfls = []*Profile{}
	prflsData := map[string]string{}

	err = json.Unmarshal(body, &prflsData)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "profile: Failed to parse auth response"),
		}
		return
	}

	for _, prflData := range prflsData {
		prfl := &Profile{}

		err = prfl.Parse(prflData)
		if err != nil {
			return
		}

		prfls = append(prfls, prfl)
	}

	err = ExportProfiles(prfls)
	if err != nil {
		return
	}

	return
}

func ExportProfiles(prfls []*Profile) (err error) {
	data, err := json.Marshal(prfls)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "profile: Failed to parse profiles"),
		}
		return
	}

	err = utils.Write(getPath(), string(data))
	if err != nil {
		return
	}

	return
}
