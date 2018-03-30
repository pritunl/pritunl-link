package sync

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	client = &http.Client{
		Timeout: 30 * time.Second,
	}
	curMod time.Time
)

type publicAddressData struct {
	Ip string `json:"ip"`
}

func SyncStates() {
	if constants.Interrupt {
		return
	}

	states := state.GetStates()
	hsh := md5.New()

	names := set.NewSet()
	for _, stat := range states {
		for i := range stat.Links {
			names.Add(fmt.Sprintf("%s-%d", stat.Id, i))
		}
		io.WriteString(hsh, stat.Hash)
	}

	newHash := hex.EncodeToString(hsh.Sum(nil))

	if newHash != state.Hash {
		ipsec.Deploy(states)
		state.Hash = newHash
	}

	resetLinks, err := state.Update(names)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"default_interface": state.GetDefaultInterface(),
			"local_address":     state.GetLocalAddress(),
			"public_address":    state.GetPublicAddress(),
			"address6":          state.GetAddress6(),
		}).Info("sync: Failed to get status")
	}

	if resetLinks != nil && len(resetLinks) != 0 {
		logrus.Warn("sync: Disconnected timeout restarting")

		err = utils.Exec("", "ipsec", "reload")
		if err != nil {
			return
		}

		for _, linkId := range resetLinks {
			utils.Exec("", "ipsec", "down", linkId)

			time.Sleep(300 * time.Millisecond)

			err = utils.Exec("", "ipsec", "up", linkId)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"link_id": linkId,
				}).Info("sync: Failed to up link")
			}
		}
	}

	return
}

func runSyncStates() {
	for {
		time.Sleep(1 * time.Second)
		SyncStates()
	}
}

func SyncDefaultIface(redeploy bool) (err error) {
	if constants.Interrupt || state.IsDirectClient {
		return
	}

	output, err := utils.ExecCombinedOutput("", "route", "-n")
	if err != nil {
		return
	}

	defaultIface := ""
	defaultGateway := ""
	outputLines := strings.Split(output, "\n")
	for _, line := range outputLines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if fields[0] == "0.0.0.0" {
			defaultIface = strings.TrimSpace(fields[len(fields)-1])
			defaultGateway = strings.TrimSpace(fields[1])
		}
	}

	if defaultIface == ipsec.DirectIface {
		return
	}

	if defaultIface != "" {
		curDefaultIface := state.GetDefaultInterface()
		state.DefaultInterface = defaultIface

		if curDefaultIface != state.GetDefaultInterface() && redeploy {
			logrus.WithFields(logrus.Fields{
				"old_default_interface": curDefaultIface,
				"default_interface":     state.GetDefaultGateway(),
			}).Info("sync: Default interface changed redeploying")

			ipsec.Redeploy()
		}
	} else if config.Config.DefaultInterface == "" {
		logrus.WithFields(logrus.Fields{
			"output": output,
		}).Warn("sync: Failed to find default interface")
	}

	if defaultGateway != "" {
		curDefaultGateway := state.GetDefaultGateway()
		state.DefaultGateway = defaultGateway

		if curDefaultGateway != state.GetDefaultGateway() && redeploy {
			logrus.WithFields(logrus.Fields{
				"old_default_gateway": curDefaultGateway,
				"default_gateway":     state.GetDefaultGateway(),
			}).Info("sync: Default gateway changed redeploying")

			ipsec.Redeploy()
		}
	} else if config.Config.DefaultGateway == "" {
		logrus.WithFields(logrus.Fields{
			"output": output,
		}).Warn("sync: Failed to find default gateway")
	}

	return
}

func runSyncDefaultIface() {
	for {
		time.Sleep(5 * time.Second)
		err := SyncDefaultIface(true)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to get default interface")
		}
	}
}

func SyncLocalAddress(redeploy bool) (err error) {
	if constants.Interrupt || state.IsDirectClient {
		return
	}

	changed := false

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
				localAddress := ipnet.IP.String()
				curLocalAddress := state.LocalAddress

				if curLocalAddress != localAddress {
					changed = true
				}
				state.LocalAddress = localAddress

				if changed && redeploy {
					logrus.WithFields(logrus.Fields{
						"old_local_address": curLocalAddress,
						"local_address":     localAddress,
					}).Info("sync: Local address changed redeploying")
				}

				break
			}
		}
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() == nil {
				address6 := ipnet.IP.String()
				curAddress6 := state.Address6

				if curAddress6 != address6 {
					changed = true
				}
				state.Address6 = address6

				if changed && redeploy {
					logrus.WithFields(logrus.Fields{
						"old_address6": curAddress6,
						"address6":     address6,
					}).Info("sync: Address6 changed redeploying")
				}

				break
			}
		}
	}

	if changed && redeploy {
		ipsec.Redeploy()
	}

	return
}

func runSyncLocalAddress() {
	for {
		time.Sleep(5 * time.Second)
		err := SyncLocalAddress(true)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to get local address")
		}
	}
}

func SyncPublicAddress(redeploy bool) (err error) {
	if constants.Interrupt || state.IsDirectClient {
		return
	}

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

	if data.Ip != "" && !state.IsDirectClient {
		publicAddress := data.Ip
		curPublicAddress := state.PublicAddress

		state.PublicAddress = publicAddress

		if curPublicAddress != publicAddress && redeploy {
			logrus.WithFields(logrus.Fields{
				"old_public_address": curPublicAddress,
				"public_address":     publicAddress,
			}).Info("sync: Public address changed redeploying")

			ipsec.Redeploy()
		}
	}

	return
}

func runSyncPublicAddress() {
	for {
		time.Sleep(30 * time.Second)
		err := SyncPublicAddress(true)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to get public address")
		}
	}
}

func SyncConfig() (err error) {
	if constants.Interrupt {
		return
	}

	mod, err := config.GetModTime()
	if err != nil {
		return
	}

	if mod != curMod {
		time.Sleep(5 * time.Second)

		mod, err = config.GetModTime()
		if err != nil {
			return
		}

		err = config.Load()
		if err != nil {
			return
		}

		logrus.Info("Reloaded config")

		curMod = mod

		ipsec.Redeploy()
	}

	return
}

func runSyncConfig() {
	curMod, _ = config.GetModTime()

	for {
		time.Sleep(500 * time.Millisecond)

		err := SyncConfig()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to sync config")
		}
	}
}

func Init() {
	err := SyncDefaultIface(false)
	if err != nil {
		time.Sleep(5 * time.Second)
		SyncDefaultIface(false)
	}
	err = SyncLocalAddress(false)
	if err != nil {
		time.Sleep(5 * time.Second)
		SyncLocalAddress(false)
	}
	err = SyncPublicAddress(false)
	if err != nil {
		time.Sleep(10 * time.Second)
		SyncPublicAddress(false)
	}
	SyncStates()
	go runSyncDefaultIface()
	go runSyncLocalAddress()
	go runSyncPublicAddress()
	go runSyncStates()
	go runSyncConfig()
}
