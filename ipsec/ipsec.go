package ipsec

import (
	"bytes"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/advertise"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/requires"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var (
	updateAdvertise bool
	deployStates    []*state.State
	curStates       []*state.State
	deployLock      sync.Mutex
	updateSleepLock sync.Mutex
	updateSleep     = constants.UpdateAdvertiseRate
)

type templateData struct {
	Id           string
	Left         string
	LeftSubnets  string
	Right        string
	RightSubnets string
	PreSharedKey string
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
		err = errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write conf"),
		}
		return
	}

	return
}

func writeTemplates(states []*state.State) (err error) {
	secretsBuf := &bytes.Buffer{}

	publicAddr := state.GetPublicAddress()
	if publicAddr == "" {
		return
	}

	for _, stat := range states {
		confBuf := &bytes.Buffer{}

		for i, link := range stat.Links {
			data := &templateData{
				Id:           fmt.Sprintf("%s-%d", stat.Id, i),
				Left:         publicAddr,
				LeftSubnets:  strings.Join(link.LeftSubnets, ","),
				Right:        link.Right,
				RightSubnets: strings.Join(link.RightSubnets, ","),
				PreSharedKey: link.PreSharedKey,
			}

			err = confTemplate.Execute(confBuf, data)
			if err != nil {
				err = errortypes.ParseError{
					errors.Wrap(err,
						"ipsec: Failed to execute conf template"),
				}
				return
			}

			err = secretsTemplate.Execute(secretsBuf, data)
			if err != nil {
				err = errortypes.ParseError{
					errors.Wrap(err,
						"ipsec: Failed to execute secrets template"),
				}
				return
			}
		}

		if stat.Type == state.DirectServer {
			clientLocal := ""
			if len(stat.Links) > 0 && len(stat.Links[0].RightSubnets) > 0 {
				clientLocal = stat.Links[0].RightSubnets[0]
			}

			// TODO
			for i := 0; i < 10; i++ {
				utils.ExecSilent(
					"",
					"iptables",
					"-t", "nat",
					"-D", "PREROUTING", "1",
				)
				utils.ExecSilent(
					"",
					"iptables",
					"-t", "nat",
					"-D", "POSTROUTING", "1",
				)
				utils.ExecSilent(
					"",
					"iptables",
					"-t", "mangle",
					"-D", "FORWARD", "1",
				)
			}

			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "PREROUTING",
				"-d", state.GetLocalAddress(),
				"-p", "udp",
				"-m", "udp",
				"--dport", "500",
				"-j", "ACCEPT",
			)
			if err != nil {
				return
			}
			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "PREROUTING",
				"-d", state.GetLocalAddress(),
				"-p", "udp",
				"-m", "udp",
				"--dport", "4500",
				"-j", "ACCEPT",
			)
			if err != nil {
				return
			}
			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "PREROUTING",
				"-d", state.GetPublicAddress(),
				"-p", "udp",
				"-m", "udp",
				"--dport", "500",
				"-j", "ACCEPT",
			)
			if err != nil {
				return
			}
			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "PREROUTING",
				"-d", state.GetPublicAddress(),
				"-p", "udp",
				"-m", "udp",
				"--dport", "4500",
				"-j", "ACCEPT",
			)
			if err != nil {
				return
			}

			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "PREROUTING",
				"-d", state.GetLocalAddress(),
				"-j", "DNAT",
				"--to-destination", clientLocal,
			)
			if err != nil {
				return
			}
			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "PREROUTING",
				"-d", state.GetPublicAddress(),
				"-j", "DNAT",
				"--to-destination", clientLocal,
			)
			if err != nil {
				return
			}

			err = utils.Exec(
				"",
				"iptables",
				"-t", "nat",
				"-A", "POSTROUTING",
				"-s", clientLocal,
				"-o", "eth0", // TODO
				"-j", "MASQUERADE",
			)
			if err != nil {
				return
			}
			err = utils.Exec(
				"",
				"iptables",
				"-t", "mangle",
				"-A", "FORWARD",
				"-s", clientLocal,
				"-p", "tcp",
				"-m", "tcp",
				"--tcp-flags", "SYN,RST SYN",
				"-j", "TCPMSS",
				"--set-mss", "1320",
			)
			if err != nil {
				return
			}
		}

		pth := path.Join(constants.IpsecDirPath,
			fmt.Sprintf("%s.conf", stat.Id))
		err = ioutil.WriteFile(pth, confBuf.Bytes(), 0644)
		if err != nil {
			err = errortypes.WriteError{
				errors.Wrap(err, "ipsec: Failed to write state conf"),
			}
			return
		}
	}

	err = ioutil.WriteFile(
		constants.IpsecSecretsPath, secretsBuf.Bytes(), 0600)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write state secrets"),
		}
		return
	}

	return
}

func deploy(states []*state.State) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "state: Interrupt"),
		}
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

	err = writeTemplates(states)
	if err != nil {
		return
	}

	err = advertise.Ports(states)
	if err != nil {
		return
	}

	err = utils.Exec("", "ipsec", "restart")
	if err != nil {
		return
	}

	err = advertise.Routes(states)
	if err != nil {
		return
	}

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

	logrus.WithFields(logrus.Fields{
		"local_address":  state.GetLocalAddress(),
		"public_address": state.GetPublicAddress(),
		"address6":       state.GetAddress6(),
	}).Info("state: Update advertisement")

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

func Redeploy() {
	deployLock.Lock()
	if deployStates == nil && curStates != nil {
		deployStates = curStates
	}
	deployLock.Unlock()
}

func runDeploy() {
	for {
		if deployStates != nil || updateAdvertise {
			deployLock.Lock()
			states := deployStates
			updateAd := false
			deployStates = nil
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
					update(states)
				} else {
					logrus.WithFields(logrus.Fields{
						"local_address":  state.GetLocalAddress(),
						"public_address": state.GetPublicAddress(),
						"address6":       state.GetAddress6(),
					}).Info("state: Deploying state")

					err := deploy(states)
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"error": err,
						}).Info("state: Failed to deploy state")

						time.Sleep(3 * time.Second)

						deployLock.Lock()
						if deployStates == nil {
							deployStates = states
						}
						deployLock.Unlock()
					} else {
						updateSleepLock.Lock()
						updateSleep = constants.UpdateAdvertiseReplay
						updateSleepLock.Unlock()
					}
				}
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func runUpdateAdvertise() {
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
			update(states)
		}
	}
}

func init() {
	module := requires.New("ipsec")
	module.After("logger")

	module.Handler = func() {
		go runDeploy()
		go runUpdateAdvertise()
	}
}
