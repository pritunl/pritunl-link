package sync

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/iptables"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
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
	states := state.GetStates()
	hsh := md5.New()

	for _, stat := range states {
		io.WriteString(hsh, stat.Hash)
	}

	newHash := hex.EncodeToString(hsh.Sum(nil))

	if newHash != state.Hash {
		ipsec.Deploy(states)
		state.Hash = newHash
	}

	return
}

func SyncStatus() {
	states := ipsec.GetStates()

	hasConnected, resetLinks, err := state.Update(states)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"default_interface": state.GetDefaultInterface(),
			"local_address":     state.GetLocalAddress(),
			"public_address":    state.GetPublicAddress(),
			"address6":          state.GetAddress6(),
		}).Info("sync: Failed to get status")
	}

	if resetLinks != nil && len(resetLinks) != 0 {
		if hasConnected {
			logrus.Warn("sync: Disconnected timeout resetting")

			for _, linkId := range resetLinks {
				state.IncLinkId(linkId)
			}

			ipsec.Redeploy(false)
		} else {
			logrus.Warn("sync: Disconnected timeout restarting")

			ipsec.Redeploy(true)
		}
	}

	return
}

func runSyncStates() {
	for {
		time.Sleep(1 * time.Second)
		if constants.Interrupt {
			return
		}
		SyncStates()
	}
}

func runSyncStatus() {
	for {
		time.Sleep(450 * time.Millisecond)
		if constants.Interrupt {
			return
		}
		SyncStatus()
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

			ipsec.Redeploy(true)
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

			ipsec.Redeploy(true)
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

	if changed && redeploy {
		ipsec.Redeploy(true)
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
			errors.Wrapf(err, "sync: Bad status %d code from server",
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

			ipsec.Redeploy(true)
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

func SyncPublicAddress6(redeploy bool) (err error) {
	if constants.Interrupt || state.IsDirectClient {
		return
	}

	req, err := http.NewRequest(
		"GET",
		constants.PublicIp6Server,
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "sync: Failed to get public address6"),
		}
		return
	}

	req.Header.Set("User-Agent", "pritunl-link")

	res, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "sync: Failed to get public address6"),
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "sync: Bad status %d code from server",
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
		curPublicAddress := state.Address6

		state.Address6 = publicAddress

		if curPublicAddress != publicAddress && redeploy {
			logrus.WithFields(logrus.Fields{
				"old_public_address": curPublicAddress,
				"public_address":     publicAddress,
			}).Info("sync: Public address6 changed redeploying")

			ipsec.Redeploy(true)
		}
	}

	return
}

func runSyncPublicAddress6() {
	for {
		time.Sleep(30 * time.Second)
		SyncPublicAddress6(true)
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

		ipsec.Redeploy(false)
	}

	return
}

func runSyncConfig() {
	curMod, _ = config.GetModTime()
	curFirewall := config.Config.Firewall

	for {
		time.Sleep(500 * time.Millisecond)

		err := SyncConfig()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("sync: Failed to sync config")
		}

		if curFirewall != config.Config.Firewall {
			curFirewall = config.Config.Firewall
			if config.Config.Firewall {
				iptables.ClearAcceptIpTables()
				iptables.ClearWgIpset()
				iptables.ResetFirewall()
			} else {
				iptables.ClearAcceptIpTables()
				iptables.ClearDropIpTables()
				iptables.RemoveWgIpset()
			}
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

	go func() {
		SyncPublicAddress6(false)
	}()

	err = SyncPublicAddress(false)
	if err != nil {
		time.Sleep(10 * time.Second)
		SyncPublicAddress(false)
	}

	time.Sleep(5 * time.Second)

	if !config.Config.Firewall {
		iptables.ClearAcceptIpTables()
		iptables.ClearDropIpTables()
		iptables.RemoveWgIpset()
	}

	SyncStates()
	go runSyncDefaultIface()
	go runSyncLocalAddress()
	go runSyncPublicAddress()
	go runSyncPublicAddress6()
	go runSyncStates()
	go runSyncStatus()
	go runSyncConfig()
}
