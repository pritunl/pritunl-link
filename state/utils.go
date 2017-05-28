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
	"time"
)

var (
	transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	clientInsec = &http.Client{
		Transport: transport,
		Timeout:   25 * time.Second,
	}
	clientSec = &http.Client{
		Timeout: 25 * time.Second,
	}
	stateCaches = map[string]*stateCache{}
	Hash        = ""
)

type stateData struct {
	Version       string            `json:"version"`
	PublicAddress string            `json:"public_address"`
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

	data := &stateData{
		Version:       constants.Version,
		PublicAddress: GetPublicAddress(),
		Status:        Status[uriData.User.Username()],
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
	nonce := utils.RandStr(32)

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
		client = clientInsec
	} else {
		client = clientSec
	}

	res, err := client.Do(req)
	if err != nil {
		state = getStateCache(uri)
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
	uris := config.Config.Uris
	urisSet := set.NewSet()

	for _, uri := range uris {
		urisSet.Add(uri)

		state, err := GetState(uri)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"uri":   uri,
				"error": err,
			}).Info("sync: Failed to get state")
			continue
		}

		states = append(states, state)
	}

	for uri := range stateCaches {
		if !urisSet.Contains(uri) {
			delete(stateCaches, uri)
		}
	}

	return
}

func cleanup(uri string) (err error) {
	uriData, err := url.ParseRequestURI(uri)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to parse uri"),
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
			errors.Wrap(err, "state: Request init error"),
		}
		return
	}

	hostId := uriData.User.Username()
	hostSecret, _ := uriData.User.Password()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := utils.RandStr(32)

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
		client = clientInsec
	} else {
		client = clientSec
	}

	res, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "state: Request delete error"),
		}
		return
	}
	defer res.Body.Close()

	return
}

func CleanUp() {
	uris := config.Config.Uris

	for _, uri := range uris {
		go cleanup(uri)
	}

	time.Sleep(3 * time.Second)

	return
}
