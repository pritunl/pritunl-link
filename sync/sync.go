package sync

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/status"
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

	total := 0
	for _, stat := range states {
		total += len(stat.Links)
		io.WriteString(hsh, stat.Hash)
	}

	newHash := hex.EncodeToString(hsh.Sum(nil))

	if newHash != state.Hash {
		ipsec.Deploy(states)
		state.Hash = newHash
	}

	status.Update(total)

	return
}

func runSyncStates() {
	for {
		time.Sleep(1 * time.Second)
		SyncStates()
		fmt.Println(state.Status)
	}
}

func SyncDefaultIface(redeploy bool) (err error) {
	output, err := utils.ExecCombinedOutput("", "route", "-n")
	if err != nil {
		return
	}

	defaultIface := ""
	outputLines := strings.Split(output, "\n")
	for _, line := range outputLines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if fields[0] == "0.0.0.0" {
			defaultIface = strings.TrimSpace(fields[len(fields)-1])
		}
	}

	if defaultIface != "" {
		curDefaultIface := state.DefaultInterface
		state.DefaultInterface = defaultIface

		if curDefaultIface != defaultIface && redeploy {
			logrus.WithFields(logrus.Fields{
				"old_default_interface": curDefaultIface,
				"default_interface":     defaultIface,
			}).Info("sync: Default interface changed redeploying")

			ipsec.Redeploy()
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"output": output,
		}).Warn("sync: Failed to find default interface")
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
	if constants.Interrupt {
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
				localAddress6 := ipnet.IP.String()
				curLocalAddress6 := state.LocalAddress

				if curLocalAddress6 != localAddress6 {
					changed = true
				}
				state.Address6 = localAddress6

				if changed && redeploy {
					logrus.WithFields(logrus.Fields{
						"old_local_address6": curLocalAddress6,
						"local_address6":     localAddress6,
					}).Info("sync: Local address6 changed redeploying")
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
	SyncDefaultIface(false)
	SyncLocalAddress(false)
	SyncPublicAddress(false)
	SyncStates()
	go runSyncDefaultIface()
	go runSyncLocalAddress()
	go runSyncPublicAddress()
	go runSyncStates()
	go runSyncConfig()
}
