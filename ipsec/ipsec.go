package ipsec

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/advertise"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/iptables"
	"github.com/pritunl/pritunl-link/requires"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
)

var (
	updateAdvertise bool
	deployStates    []*state.State
	deployRestart   = true
	curStates       []*state.State
	deployLock      sync.Mutex
	updateSleepLock sync.Mutex
	updateSleep     = constants.UpdateAdvertiseRate
	wgHash          = ""
)

type templateData struct {
	Id             string
	Action         string
	Left           string
	LeftWg         string
	LeftSubnets    string
	Right          string
	RightSubnets   string
	RightWg        string
	PreSharedKey   string
	WgHash         string
	WgPort         int
	WgPreSharedKey string
	WgPublicKey    string
	WgPrivateKey   string
	IkeCiphers     string
	EspCiphers     string
}

func putIpTables(stat *state.State) (err error) {
	clientLocal := ""
	if len(stat.Links) > 0 && len(stat.Links[0].RightSubnets) > 0 {
		clientLocal = stat.Links[0].RightSubnets[0]
	}
	clientLocal = strings.SplitN(clientLocal, "/", 2)[0]

	localAddress := state.GetLocalAddress()
	publicAddress := state.GetPublicAddress()
	defaultIface := state.GetDefaultInterface()
	directMode := GetDirectMode()

	directIp, err := GetDirectClientIp()
	if err != nil {
		return
	}

	directClientIp := directIp.String()

	if clientLocal == "" || localAddress == "" ||
		publicAddress == "" || defaultIface == "" {

		logrus.WithFields(logrus.Fields{
			"client_local_address": clientLocal,
			"local_address":        localAddress,
			"public_address":       publicAddress,
			"default_interface":    defaultIface,
		}).Warn("ipsec: Missing required values for iptables")

		return
	}

	err = iptables.UpsertRule(
		"nat",
		"PREROUTING",
		"-d", localAddress,
		"-p", "udp",
		"-m", "udp",
		"--dport", "500",
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}
	err = iptables.UpsertRule(
		"nat",
		"PREROUTING",
		"-d", localAddress,
		"-p", "udp",
		"-m", "udp",
		"--dport", "4500",
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}
	if stat.Protocol == "wg" && stat.WgPort != 0 {
		err = iptables.UpsertRule(
			"nat",
			"PREROUTING",
			"-d", localAddress,
			"-p", "udp",
			"-m", "udp",
			"--dport", strconv.Itoa(stat.WgPort),
			"-j", "ACCEPT",
			"-m", "comment",
			"--comment", "pritunl-link-direct",
		)
		if err != nil {
			return
		}
	}
	err = iptables.UpsertRule(
		"nat",
		"PREROUTING",
		"-d", publicAddress,
		"-p", "udp",
		"-m", "udp",
		"--dport", "500",
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}
	err = iptables.UpsertRule(
		"nat",
		"PREROUTING",
		"-d", publicAddress,
		"-p", "udp",
		"-m", "udp",
		"--dport", "4500",
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}
	if stat.Protocol == "wg" && stat.WgPort != 0 {
		err = iptables.UpsertRule(
			"nat",
			"PREROUTING",
			"-d", publicAddress,
			"-p", "udp",
			"-m", "udp",
			"--dport", strconv.Itoa(stat.WgPort),
			"-j", "ACCEPT",
			"-m", "comment",
			"--comment", "pritunl-link-direct",
		)
		if err != nil {
			return
		}
	}
	if config.Config.DirectSsh {
		err = iptables.UpsertRule(
			"nat",
			"PREROUTING",
			"-d", localAddress,
			"-p", "tcp",
			"-m", "tcp",
			"--dport", "22",
			"-j", "ACCEPT",
			"-m", "comment",
			"--comment", "pritunl-link-direct",
		)
		if err != nil {
			return
		}
		err = iptables.UpsertRule(
			"nat",
			"PREROUTING",
			"-d", publicAddress,
			"-p", "tcp",
			"-m", "tcp",
			"--dport", "22",
			"-j", "ACCEPT",
			"-m", "comment",
			"--comment", "pritunl-link-direct",
		)
		if err != nil {
			return
		}
	} else {
		iptables.DeleteRule(
			"nat",
			"PREROUTING",
			"-d", localAddress,
			"-p", "tcp",
			"-m", "tcp",
			"--dport", "22",
			"-j", "ACCEPT",
			"-m", "comment",
			"--comment", "pritunl-link-direct",
		)
		iptables.DeleteRule(
			"nat",
			"PREROUTING",
			"-d", publicAddress,
			"-p", "tcp",
			"-m", "tcp",
			"--dport", "22",
			"-j", "ACCEPT",
			"-m", "comment",
			"--comment", "pritunl-link-direct",
		)
	}

	directSource := directClientIp
	if directMode == DirectPolicy {
		directSource = clientLocal
	}

	err = iptables.UpsertRule(
		"nat",
		"PREROUTING",
		"-d", localAddress,
		"-j", "DNAT",
		"--to-destination", directSource,
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}
	err = iptables.UpsertRule(
		"nat",
		"PREROUTING",
		"-d", publicAddress,
		"-j", "DNAT",
		"--to-destination", directSource,
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}

	err = iptables.UpsertRule(
		"nat",
		"POSTROUTING",
		"-s", directSource+"/32",
		"-o", defaultIface,
		"-j", "MASQUERADE",
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}

	err = iptables.UpsertRule(
		"mangle",
		"FORWARD",
		"-s", directSource+"/32",
		"-p", "tcp",
		"-m", "tcp",
		"--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS",
		"--set-mss", "1320",
		"-m", "comment",
		"--comment", "pritunl-link-direct",
	)
	if err != nil {
		return
	}

	return
}

func clearDir() (err error) {
	err = os.RemoveAll(constants.IpsecDirPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "ipsec: Failed to remove ipsec conf dir"),
		}
		return
	}

	err = os.MkdirAll(constants.IpsecDirPath, 0755)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "ipsec: Failed to create ipsec conf dir"),
		}
		return
	}

	return
}

func writeConf() (err error) {
	data := fmt.Sprintf("include %s/*.conf", constants.IpsecDirPath)

	pth := path.Join(constants.IpsecConfPath)

	curData, _ := ioutil.ReadFile(pth)
	if curData != nil {
		if strings.Contains(string(curData), data) {
			return
		}
	}

	err = ioutil.WriteFile(pth, []byte(data), 0644)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write conf"),
		}
		return
	}

	return
}

func writeTemplates(states []*state.State) (iptablesState bool, err error) {
	secretsBuf := &bytes.Buffer{}

	publicAddr := state.GetPublicAddress()
	publicAddr6 := state.GetAddress6()

	confs := map[string]*bytes.Buffer{}

	for _, stat := range states {
		if stat.Protocol != "" && stat.Protocol != "ipsec" {
			continue
		}

		confBuf := &bytes.Buffer{}

		for _, link := range stat.Links {
			leftSubnets := strings.Join(link.LeftSubnets, ",")
			rightSubnets := strings.Join(link.RightSubnets, ",")

			if GetDirectMode() == DirectPolicy {
				if stat.Type == state.DirectServer {
					leftSubnets = "0.0.0.0/0"
				} else if stat.Type == state.DirectClient {
					rightSubnets = "0.0.0.0/0"
				}
			}

			left := ""
			if stat.Ipv6 {
				left = publicAddr6
			} else {
				left = publicAddr
			}

			action := "restart"
			if stat.Action != "" {
				action = stat.Action
			}

			ikeCiphersData := ""
			if stat.PreferredIke != "" {
				if stat.ForcePreferred {
					ikeCiphersData = stat.PreferredIke
				} else {
					ikeCiphersData = stat.PreferredIke + "," + ikeCiphers
				}
			} else {
				ikeCiphersData = ikeCiphers
			}

			espCiphersData := ""
			if stat.PreferredEsp != "" {
				if stat.ForcePreferred {
					espCiphersData = stat.PreferredEsp
				} else {
					espCiphersData = stat.PreferredEsp + "," + espCiphers
				}
			} else {
				espCiphersData = espCiphers
			}

			data := &templateData{
				Id:           state.GetLinkId(stat.Id, link.Id, link.Hash),
				Action:       action,
				Left:         left,
				LeftSubnets:  leftSubnets,
				Right:        link.Right,
				RightSubnets: rightSubnets,
				PreSharedKey: link.PreSharedKey,
				IkeCiphers:   ikeCiphersData,
				EspCiphers:   espCiphersData,
			}

			err = confTemplate.Execute(confBuf, data)
			if err != nil {
				err = &errortypes.ParseError{
					errors.Wrap(err,
						"ipsec: Failed to execute conf template"),
				}
				return
			}

			if config.Config.CustomOptions != nil {
				for _, opt := range config.Config.CustomOptions {
					_, err = confBuf.WriteString("	" + opt + "\n")
					if err != nil {
						err = &errortypes.WriteError{
							errors.Wrap(err, "ipsec: Failed to "+
								"write custom option"),
						}
						return
					}
				}
			}

			if link.Static && (len(link.LeftSubnets) > 1 ||
				len(link.RightSubnets) > 1) {

				for x, leftSubnet := range link.LeftSubnets {
					for y, rightSubnet := range link.RightSubnets {
						data := &templateData{
							Id: state.GetLinkIds(
								stat.Id, link.Id, x, y, link.Hash),
							Action:       action,
							Left:         left,
							LeftSubnets:  leftSubnet,
							Right:        link.Right,
							RightSubnets: rightSubnet,
							PreSharedKey: link.PreSharedKey,
							IkeCiphers:   ikeCiphersData,
							EspCiphers:   espCiphersData,
						}

						err = confTemplate.Execute(confBuf, data)
						if err != nil {
							err = &errortypes.ParseError{
								errors.Wrap(err,
									"ipsec: Failed to execute conf template"),
							}
							return
						}

						if config.Config.CustomOptions != nil {
							for _, opt := range config.Config.CustomOptions {
								_, err = confBuf.WriteString("	" + opt + "\n")
								if err != nil {
									err = &errortypes.WriteError{
										errors.Wrap(err, "ipsec: Failed to "+
											"write custom option"),
									}
									return
								}
							}
						}
					}
				}
			}

			err = secretsTemplate.Execute(secretsBuf, data)
			if err != nil {
				err = &errortypes.ParseError{
					errors.Wrap(err,
						"ipsec: Failed to execute secrets template"),
				}
				return
			}
		}

		if stat.Type == state.DirectServer && len(stat.Links) != 0 {
			iptablesState = true

			err = putIpTables(stat)
			if err != nil {
				return
			}
		}

		pth := path.Join(constants.IpsecDirPath,
			fmt.Sprintf("%s.conf", stat.Id))
		confs[pth] = confBuf
	}

	err = ioutil.WriteFile(
		constants.IpsecSecretsPath, secretsBuf.Bytes(), 0600)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write state secrets"),
		}
		return
	}

	for pth, confBuf := range confs {
		err = ioutil.WriteFile(pth, confBuf.Bytes(), 0644)
		if err != nil {
			err = &errortypes.WriteError{
				errors.Wrap(err, "ipsec: Failed to write state conf"),
			}
			return
		}
	}

	if !iptablesState {
		err = iptables.ClearIpTables()
		if err != nil {
			return
		}
	}

	return
}

func writeWgTemplates(states []*state.State) (wgIfaces, modWgIfaces []string,
	iptablesState bool, err error) {

	publicAddr := state.GetPublicAddress()
	publicAddr6 := state.GetAddress6()

	confs := map[string]*bytes.Buffer{}

	for _, stat := range states {
		if stat.Protocol != "wg" {
			continue
		}

		confBuf := &bytes.Buffer{}

		privKey, e := config.State.GetPrivateKey(stat.Id)
		if e != nil {
			err = e
			return
		}

		data := &templateData{
			WgHash:       wgHash,
			WgPort:       stat.WgPort,
			WgPrivateKey: privKey,
		}

		err = confWgTemplate.Execute(confBuf, data)
		if err != nil {
			err = &errortypes.ParseError{
				errors.Wrap(err,
					"ipsec: Failed to execute conf template"),
			}
			return
		}

		for _, link := range stat.Links {
			leftSubnets := strings.Join(link.LeftSubnets, ",")
			rightSubnets := strings.Join(link.RightSubnets, ",")

			if link.WgPublicKey == "" {
				continue
			}

			if GetDirectMode() == DirectPolicy {
				if stat.Type == state.DirectServer {
					leftSubnets = "0.0.0.0/0"
				} else if stat.Type == state.DirectClient {
					rightSubnets = "0.0.0.0/0"
				}
			}

			left := ""
			if stat.Ipv6 {
				left = publicAddr6
			} else {
				left = publicAddr
			}

			data := &templateData{
				Id:             state.GetLinkId(stat.Id, link.Id, link.Hash),
				Left:           left,
				LeftWg:         utils.FormatHost(left),
				LeftSubnets:    leftSubnets,
				Right:          link.Right,
				RightWg:        utils.FormatHost(link.Right),
				RightSubnets:   rightSubnets,
				WgPort:         stat.WgPort,
				WgPublicKey:    link.WgPublicKey,
				WgPreSharedKey: PreSharedKeyToWg(link.PreSharedKey),
			}

			err = confWgPeerTemplate.Execute(confBuf, data)
			if err != nil {
				err = &errortypes.ParseError{
					errors.Wrap(err,
						"ipsec: Failed to execute conf template"),
				}
				return
			}
		}

		if stat.Type == state.DirectServer && len(stat.Links) != 0 {
			iptablesState = true

			err = putIpTables(stat)
			if err != nil {
				return
			}
		}

		iface := GetWgIface(stat.Id)
		wgIfaces = append(wgIfaces, iface)
		confs[iface] = confBuf
	}

	for iface, confBuf := range confs {
		pth := path.Join(constants.WgDirPath, fmt.Sprintf("%s.conf", iface))

		curData, _ := ioutil.ReadFile(pth)
		newData := confBuf.Bytes()

		if !bytes.Equal(curData, newData) {
			modWgIfaces = append(modWgIfaces, iface)
			err = ioutil.WriteFile(pth, confBuf.Bytes(), 0600)
			if err != nil {
				err = &errortypes.WriteError{
					errors.Wrap(err, "ipsec: Failed to write state conf"),
				}
				return
			}
		}
	}

	return
}

func deploy(states []*state.State, restart bool) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "state: Interrupt"),
		}
		return
	}

	isDirect := false
	for _, stat := range states {
		if (stat.Type == state.DirectClient ||
			stat.Type == state.DirectServer) && len(stat.Links) != 0 {

			err = StartTunnel(stat)
			if err != nil {
				return
			}

			isDirect = true

			if stat.Type == state.DirectClient {
				state.DirectIpsecState = stat
			}

			break
		}
	}

	if !isDirect {
		StopTunnel()
		state.DirectIpsecState = nil
	}

	err = iptables.ClearIpTables()
	if err != nil {
		return
	}

	err = utils.NetInit()
	if err != nil {
		return
	}

	err = clearDir()
	if err != nil {
		return
	}

	err = writeConf()
	if err != nil {
		return
	}

	iptablesState, err := writeTemplates(states)
	if err != nil {
		return
	}

	wgIfaces, modWgIfaces, iptablesWgState, err := writeWgTemplates(states)
	if err != nil {
		return
	}

	if !iptablesState && !iptablesWgState {
		err = iptables.ClearIpTables()
		if err != nil {
			return
		}
	}

	err = advertise.Ports(states)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("state: Failed to advertise ports")
		err = nil
	}

	time.Sleep(200 * time.Millisecond)

	curWgIfaces, activeWgIface, err := GetWgIfaces()
	if err != nil {
		return
	}

	newWgIfaces := set.NewSet()
	for _, wgIface := range wgIfaces {
		newWgIfaces.Add(wgIface)
	}

	for _, wgIface := range modWgIfaces {
		confPth := path.Join(constants.WgDirPath,
			fmt.Sprintf("%s.conf", wgIface))

		_ = utils.Exec("", "wg-quick", "down", wgIface)
		err = utils.Exec("", "wg-quick", "up", wgIface)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"wg_iface": wgIface,
				"wg_conf":  confPth,
				"error":    err,
			}).Error("state: Error bringing up wg conf")
			err = nil
		}
	}

	delWgIfaces := curWgIfaces.Copy()
	delWgIfaces.Subtract(newWgIfaces)
	for ifaceInf := range delWgIfaces.Iter() {
		iface := ifaceInf.(string)
		confPth := path.Join(constants.WgDirPath,
			fmt.Sprintf("%s.conf", iface))

		err = utils.Exec("", "wg-quick", "down", iface)
		if err != nil {
			if activeWgIface.Contains(iface) {
				logrus.WithFields(logrus.Fields{
					"wg_iface": iface,
					"error":    err,
				}).Error("state: Error bringing down wg conf")
			}
			err = nil
		}

		os.Remove(confPth)
	}

	if restart {
		err = utils.Exec("", "ipsec", "restart")
		if err != nil {
			return
		}
	} else {
		unknownIds, e := state.Unknown(states)
		if e != nil {
			err = e
			return
		}

		if unknownIds != nil && len(unknownIds) > 0 {
			for _, linkId := range unknownIds {
				logrus.WithFields(logrus.Fields{
					"link_id": linkId,
				}).Info("state: Stopping removed link")
				go Shutdown(linkId)
			}

			time.Sleep(3 * time.Second)
		}

		//err = utils.Exec("", "ipsec", "rereadsecrets")
		//if err != nil {
		//	return
		//}

		err = utils.Exec("", "ipsec", "rereadall")
		if err != nil {
			return
		}

		time.Sleep(100 * time.Millisecond)

		err = utils.Exec("", "ipsec", "update")
		if err != nil {
			return
		}
	}

	err = advertise.Routes(states)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("state: Failed to advertise routes")
		err = nil
	}

	isDirectClient := false
	for _, stat := range states {
		if stat.Type == state.DirectClient && len(stat.Links) != 0 {
			isDirectClient = true
			break
		}
	}
	state.IsDirectClient = isDirectClient

	return
}

func update(states []*state.State) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "state: Interrupt"),
		}
		return
	}

	if config.Config.DisableAdvertiseUpdate {
		return
	}

	hasLinks := false
	for _, ste := range states {
		if ste.Links != nil && len(ste.Links) != 0 {
			hasLinks = true
		}
	}

	if !hasLinks {
		return
	}

	err = advertise.Ports(states)
	if err != nil {
		return
	}

	err = advertise.Routes(states)
	if err != nil {
		return
	}

	return
}

func Deploy(states []*state.State) {
	deployLock.Lock()
	deployStates = states
	deployLock.Unlock()
}

func Redeploy(restart bool) {
	deployLock.Lock()
	if deployStates == nil && curStates != nil {
		deployStates = curStates
	}
	if restart {
		deployRestart = true
	}
	deployLock.Unlock()
}

func GetStates() []*state.State {
	return curStates
}

func runDeploy() {
	for {
		if deployStates != nil || updateAdvertise {
			deployLock.Lock()
			states := deployStates
			restart := deployRestart
			updateAd := false
			deployStates = nil
			deployRestart = false
			if states != nil {
				curStates = states
			} else if updateAdvertise {
				updateAd = true
				states = curStates
			}
			updateAdvertise = false
			deployLock.Unlock()

			if states != nil {
				if updateAd {
					err := update(states)
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"error": err,
						}).Error(
							"state: Failed to update route advertisement")
					}
				} else {
					logrus.WithFields(logrus.Fields{
						"default_interface": state.GetDefaultInterface(),
						"local_address":     state.GetLocalAddress(),
						"public_address":    state.GetPublicAddress(),
						"address6":          state.GetAddress6(),
						"states_len":        len(states),
					}).Info("state: Deploying state")

					err := deploy(states, restart)
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"error": err,
						}).Error("state: Failed to deploy state")

						time.Sleep(3 * time.Second)

						deployLock.Lock()
						if deployStates == nil {
							deployStates = states
						}
						if restart {
							deployRestart = true
						}
						deployLock.Unlock()

						time.Sleep(200 * time.Millisecond)
						continue
					}

					updateSleepLock.Lock()
					updateSleep = constants.UpdateAdvertiseReplay
					updateSleepLock.Unlock()
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func runUpdateAdvertise() {
	defer func() {
		r := recover()
		err := &errortypes.UnknownError{
			errors.New("ipsec: Route advertisement panic"),
		}

		logrus.WithFields(logrus.Fields{
			"panic": r,
			"error": err,
		}).Error("ipsec: Panic in route advertisement, restarting...")

		time.Sleep(3 * time.Second)
		go runUpdateAdvertise()
	}()

	for {
		for {
			time.Sleep(1 * time.Second)

			updateSleepLock.Lock()
			updateSleep -= 1
			if updateSleep <= 0 {
				updateSleep = constants.UpdateAdvertiseRate
				updateSleepLock.Unlock()
				break
			} else {
				updateSleepLock.Unlock()
			}
		}

		states := curStates
		if states != nil {
			err := update(states)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Error("ipsec: Failed to update route advertisement")
			}
		}
	}
}

func init() {
	module := requires.New("ipsec")
	module.After("logger")

	module.Handler = func() (err error) {
		wgHash, err = utils.RandStr(32)
		if err != nil {
			return
		}

		go runDeploy()
		go runUpdateAdvertise()
		go runRoutes()

		return
	}
}
