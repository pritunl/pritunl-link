package sync

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/status"
	"io"
	"net"
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout: 2 * time.Second,
}

type publicAddressData struct {
	Ip string `json:"ip"`
}

func SyncStates() {
	states := state.GetStates()
	hsh := md5.New()

	for _, stat := range states {
		io.WriteString(hsh, stat.Hash)
	}

	newHash := hex.EncodeToString(hsh.Sum(nil))

	if newHash != state.Hash {
		logrus.Info("state: Deploying state")

		state.States = states

		err := ipsec.Deploy()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("state: Failed to deploy state")
			time.Sleep(1 * time.Second)
			return
		}

		state.Hash = newHash
	}
}

func runSyncStates() {
	for {
		time.Sleep(1 * time.Second)
		SyncStates()
		status.Update()
		fmt.Println(status.Status)
	}
}

func SyncLocalAddress() (err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "sync: Failed to get interface addresses"),
		}
		return
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				state.LocalAddress = ipnet.IP.String()
				return
			}
		}
	}

	return
}

func runSyncLocalAddress() {
	for {
		time.Sleep(5 * time.Second)
		err := SyncPublicAddress()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to get local address")
			return
		}
	}
}

func SyncPublicAddress() (err error) {
	req, err := http.NewRequest(
		"GET",
		constants.PublicIpServer,
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "sync: Failed to get public address"),
		}
		return
	}

	req.Header.Set("User-Agent", "pritunl-link")

	res, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "sync: Failed to get public address"),
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "sync: Bad status %n code from server",
				res.StatusCode),
		}
		return
	}

	data := &publicAddressData{}

	err = json.NewDecoder(res.Body).Decode(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "sync: Failed to parse data"),
		}
		return
	}

	if data.Ip != "" {
		state.PublicAddress = data.Ip
	}

	return
}

func runSyncPublicAddress() {
	for {
		time.Sleep(30 * time.Second)
		err := SyncPublicAddress()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to get public address")
			return
		}
	}
}

func Init() {
	SyncLocalAddress()
	SyncPublicAddress()
	SyncStates()
	go runSyncLocalAddress()
	go runSyncPublicAddress()
	go runSyncStates()
}
