package state

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/utils"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	transport = &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	ClientInsec = &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	ClientSec = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: 10 * time.Second,
	}
	stateCaches = map[string]*stateCache{}
	Hash        = ""
)

type stateData struct {
	Version       string            `json:"version"`
	PublicAddress string            `json:"public_address"`
	LocalAddress  string            `json:"local_address"`
	Address6      string            `json:"address6"`
	Status        map[string]string `json:"status"`
	Errors        []string          `json:"errors"`
}

type stateCache struct {
	Timestamp time.Time
	State     *State
}

func decResp(secret, iv, sig, encData string) (cipData []byte, err error) {
	hashFunc := hmac.New(sha512.New, []byte(secret))
	hashFunc.Write([]byte(encData))
	rawSignature := hashFunc.Sum(nil)
	testSig := base64.StdEncoding.EncodeToString(rawSignature)
	if sig != testSig {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Cipher data signature invalid"),
		}
		return
	}

	cipIv, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to decode cipher IV"),
		}
		return
	}

	encKeyHash := sha256.New()
	encKeyHash.Write([]byte(secret))
	cipKey := encKeyHash.Sum(nil)

	cipData, err = base64.StdEncoding.DecodeString(encData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to decode response data"),
		}
		return
	}

	if len(cipIv) != aes.BlockSize {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Invalid cipher key"),
		}
		return
	}

	if len(cipData)%aes.BlockSize != 0 {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Invalid cipher data"),
		}
		return
	}

	block, err := aes.NewCipher(cipKey)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to load cipher"),
		}
		return
	}

	mode := cipher.NewCBCDecrypter(block, cipIv)
	mode.CryptBlocks(cipData, cipData)

	cipData = bytes.TrimRight(cipData, "\x00")

	return
}

func getStateCache(uri string) (state *State) {
	cache, ok := stateCaches[uri]
	if ok && time.Since(cache.Timestamp) < constants.StateCacheTtl {
		state = cache.State
		return
	}

	return
}

func GetState(uri string) (state *State, err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "state: Interrupt"),
		}
		return
	}

	state = &State{}

	uriData, err := url.ParseRequestURI(uri)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to parse uri"),
		}
		return
	}

	status := Status
	stateId := uriData.User.Username()
	stateStatus := map[string]string{}

	for connId, connStatus := range status {
		connIds := strings.Split(connId, "-")
		if len(connIds) != 3 {
			continue
		}

		if connIds[0] != stateId {
			continue
		}

		curStatus := stateStatus[connIds[1]]
		if curStatus == "" || curStatus == "disconnected" ||
			(curStatus == "connecting" && connStatus == "connected") {

			stateStatus[connIds[1]] = connStatus
		}
	}

	data := &stateData{
		Version:       constants.Version,
		PublicAddress: GetPublicAddress(),
		LocalAddress:  GetLocalAddress(),
		Address6:      GetAddress6(),
		Status:        stateStatus,
	}
	dataBuf := &bytes.Buffer{}

	err = json.NewEncoder(dataBuf).Encode(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to parse request data"),
		}
		return
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://%s/link/state", uriData.Host),
		dataBuf,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "state: Request init error"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")

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
		"PUT",
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
		client = ClientInsec
	} else {
		client = ClientSec
	}

	start := time.Now()

	res, err := client.Do(req)
	if err != nil {
		state = getStateCache(uri)

		logrus.WithFields(logrus.Fields{
			"duration":  utils.ToFixed(time.Since(start).Seconds(), 2),
			"has_cache": state != nil,
			"error":     err,
		}).Warn("state: Request failed")

		if state == nil {
			err = &errortypes.RequestError{
				errors.Wrap(err, "state: Request put error"),
			}
		} else {
			err = nil
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode >= 500 && res.StatusCode < 600 {
		state = getStateCache(uri)
		if state == nil {
			err = &errortypes.RequestError{
				errors.Wrapf(err, "state: Bad status %n code from server",
					res.StatusCode),
			}
		} else {
			err = nil
		}
		return
	} else if res.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "state: Bad status %n code from server",
				res.StatusCode),
		}
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "state: Failed to read response body"),
		}
		return
	}

	decBody, err := decResp(
		hostSecret,
		res.Header.Get("Cipher-IV"),
		res.Header.Get("Cipher-Signature"),
		string(body),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "state: Failed to decrypt response"),
		}
		return
	}

	err = json.Unmarshal(decBody, state)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to unmarshal data"),
		}
		return
	}

	cache := &stateCache{
		Timestamp: time.Now(),
		State:     state,
	}
	stateCaches[uri] = cache

	return
}

func GetStates() (states []*State) {
	states = []*State{}
	statesMap := map[int]*State{}
	uris := config.Config.Uris
	urisSet := set.NewSet()
	waiter := sync.WaitGroup{}

	for i, uri := range uris {
		urisSet.Add(uri)
		waiter.Add(1)

		go func(i int, uri string) {
			state, err := GetState(uri)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"uri":   uri,
					"error": err,
				}).Info("state: Failed to get state")
				statesMap[i] = nil
			} else {
				statesMap[i] = state
			}

			waiter.Done()
		}(i, uri)
	}

	waiter.Wait()

	for i := range uris {
		state := statesMap[i]

		if state != nil {
			states = append(states, state)
		}
	}

	for uri := range stateCaches {
		if !urisSet.Contains(uri) {
			delete(stateCaches, uri)
		}
	}

	return
}
