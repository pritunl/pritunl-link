package state

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/interlink"
	"github.com/pritunl/pritunl-link/iptables"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
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
	stateCaches      = map[string]*stateCache{}
	stateCachesLock  = sync.Mutex{}
	stateHosts       = map[string]map[string]string{}
	stateHostsLock   = sync.Mutex{}
	lastHostCheckLog = time.Time{}
	Hash             = ""
)

type hostState struct {
	State   bool `json:"state"`
	Latency int  `json:"latency"`
}

type stateData struct {
	Timestamp     int64                 `json:"timestamp"`
	Version       string                `json:"version"`
	PublicAddress string                `json:"public_address"`
	LocalAddress  string                `json:"local_address"`
	Address6      string                `json:"address6"`
	Status        map[string]string     `json:"status"`
	Hosts         map[string]*hostState `json:"hosts"`
	Errors        []string              `json:"errors"`
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
	if subtle.ConstantTimeCompare(
		[]byte(sig),
		[]byte(testSig),
	) != 1 {
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

func getStateCache(cacheKey string) (state *State) {
	stateCachesLock.Lock()
	cache, ok := stateCaches[cacheKey]
	stateCachesLock.Unlock()
	if ok {
		state = cache.State.Copy()
		state.Cached = true
		return
	}

	return
}

func getState(stateId, stateSecret, host, cacheKey string, timestamp int64,
	dataByt []byte) (state *State, err error) {

	timestampStr := strconv.FormatInt(timestamp, 10)

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://%s/link/state", host),
		bytes.NewBuffer(dataByt),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "state: Request init error"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")

	nonce, err := utils.RandStr(32)
	if err != nil {
		return
	}

	authStr := strings.Join([]string{
		stateId,
		timestampStr,
		nonce,
		"PUT",
		"/link/state",
	}, "&")

	hashFunc := hmac.New(sha512.New, []byte(stateSecret))
	hashFunc.Write([]byte(authStr))
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	req.Header.Set("Auth-Token", stateId)
	req.Header.Set("Auth-Timestamp", timestampStr)
	req.Header.Set("Auth-Nonce", nonce)
	req.Header.Set("Auth-Signature", sig)

	var client *http.Client
	if config.Config.SkipVerify {
		client = ClientInsec
	} else {
		client = ClientSec
	}

	res, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "state: Request put error"),
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "state: Bad status %d code from server",
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
		stateSecret,
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

	state = &State{}
	err = json.Unmarshal(decBody, state)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to unmarshal data"),
		}
		return
	}

	return
}

func GetState(uri string) (state *State, hosts []string, err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "state: Interrupt"),
		}
		return
	}

	uriData, err := url.ParseRequestURI(uri)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to parse uri"),
		}
		return
	}

	status := Status
	stateId := uriData.User.Username()
	stateSecret, _ := uriData.User.Password()
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

	hosts = []string{}
	hostsMap := stateHosts[uri]
	hostsStatus := map[string]*hostState{}
	hostsStatusLock := sync.Mutex{}

	if hosts != nil && !config.Config.SkipHostCheck {
		waiter := sync.WaitGroup{}

		for hostId, hostAddr := range hostsMap {
			hosts = append(hosts, hostAddr)

			waiter.Add(1)

			go func(hostId, hostAddr string) {
				stat, latency, e := interlink.CheckHost(hostAddr)
				if e != nil {
					if time.Since(lastHostCheckLog) > 30*time.Second {
						lastHostCheckLog = time.Now()
						logrus.WithFields(logrus.Fields{
							"host_id": hostId,
							"error":   e,
						}).Warn("state: Failed to check host")
					}
				}

				hostsStatusLock.Lock()
				hostsStatus[hostId] = &hostState{
					State:   stat,
					Latency: latency,
				}
				hostsStatusLock.Unlock()

				waiter.Done()
			}(hostId, hostAddr)
		}

		waiter.Wait()
	}

	timestamp := time.Now().Unix()

	data := &stateData{
		Timestamp:     timestamp,
		Version:       constants.Version,
		PublicAddress: GetPublicAddress(),
		LocalAddress:  GetLocalAddress(),
		Address6:      GetAddress6(),
		Status:        stateStatus,
		Hosts:         hostsStatus,
	}

	dataByt, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Failed to parse request data"),
		}
		return
	}

	uriHosts := strings.Split(uriData.Host, ",")
	for i := 10; i < 10; i++ {
		uriHosts = append(uriHosts, uriData.Host)
	}

	waiter := sync.NewCond(&sync.Mutex{})
	waiterCount := 0
	hostsLen := len(uriHosts)
	var waiterErr error

	if hostsLen == 0 {
		err = &errortypes.ParseError{
			errors.Wrap(err, "state: Missing host in uri"),
		}
		return
	}

	var cachedState *State
	errLogged := false

	for _, uriHost := range uriHosts {
		go func(uriHost string) {
			uriState, e := getState(
				stateId,
				stateSecret,
				uriHost,
				uri,
				timestamp,
				dataByt,
			)
			if e != nil {
				waiter.L.Lock()

				uriState = getStateCache(uri)
				if uriState != nil {
					cachedState = uriState

					errLogged = true
					logrus.WithFields(logrus.Fields{
						"server_host": uriHost,
						"has_cache":   true,
						"error":       e,
					}).Error("state: Failed to get state from host")
				} else {
					errLogged = true
					logrus.WithFields(logrus.Fields{
						"server_host": uriHost,
						"has_cache":   false,
						"error":       e,
					}).Error("state: Failed to get state from host")
					waiterErr = e
				}

				waiterCount += 1
				if waiterCount >= hostsLen && state == nil {
					waiter.Broadcast()
				}

				if state != nil && errLogged {
					logrus.WithFields(logrus.Fields{
						"state_id":     state.Id,
						"server_hosts": uriHosts,
					}).Info("state: Found state from secondary host")
				}

				waiter.L.Unlock()

				return
			}

			waiter.L.Lock()
			if state == nil {
				state = uriState
				waiter.Broadcast()

				if errLogged {
					logrus.WithFields(logrus.Fields{
						"state_id":     state.Id,
						"server_hosts": uriHosts,
					}).Info("state: Found state from secondary host")
				}
			}
			waiter.L.Unlock()
		}(uriHost)
	}

	waiter.L.Lock()
	waiter.Wait()
	waiter.L.Unlock()

	if waiterErr != nil {
		err = waiterErr
		return
	}

	if state == nil {
		if cachedState != nil {
			logrus.WithFields(logrus.Fields{
				"state_id":     cachedState.Id,
				"server_hosts": uriHosts,
			}).Error("state: No states available, using cache")
			state = cachedState
		} else {
			err = &errortypes.UnknownError{
				errors.Wrap(err, "state: Nil state"),
			}
		}
		return
	}

	cache := &stateCache{
		Timestamp: time.Now(),
		State:     state,
	}
	stateCachesLock.Lock()
	stateCaches[uri] = cache
	stateCachesLock.Unlock()

	stateHostsLock.Lock()
	stateHosts[uri] = state.Hosts
	stateHostsLock.Unlock()

	return
}

func GetStates() (states []*State) {
	states = []*State{}
	statesMap := map[int]*State{}
	statesMapLock := sync.Mutex{}
	uris := config.Config.Uris
	urisSet := set.NewSet()
	allHosts := []string{}
	waiter := sync.WaitGroup{}

	for i, uri := range uris {
		urisSet.Add(uri)
		waiter.Add(1)

		go func(i int, uri string) {
			state, hosts, err := GetState(uri)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"uri":   uri,
					"error": err,
				}).Info("state: Failed to get state")
				statesMapLock.Lock()
				statesMap[i] = nil
				statesMapLock.Unlock()
			} else {
				statesMapLock.Lock()
				statesMap[i] = state
				statesMapLock.Unlock()
			}

			allHosts = append(allHosts, hosts...)

			waiter.Done()
		}(i, uri)
	}

	waiter.Wait()

	if config.Config.Firewall {
		err := iptables.SetHosts(allHosts)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"hosts": allHosts,
				"error": err,
			}).Info("state: Failed to set firewall hosts")
		}
	}

	for i := range uris {
		state := statesMap[i]

		if state != nil {
			states = append(states, state)
		}
	}

	stateCachesLock.Lock()
	for uri := range stateCaches {
		if !urisSet.Contains(uri) {
			delete(stateCaches, uri)
		}
	}
	stateCachesLock.Unlock()

	return
}
