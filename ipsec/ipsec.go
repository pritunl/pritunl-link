package ipsec

import (
	"bytes"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/advertise"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/requires"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"time"
)

var (
	deployStates []*state.State
	curStates    []*state.State
	deployLock   sync.Mutex
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
	err = utils.RemoveAll(config.Config.IpsecDirPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "ipsec: Failed to remove ipsec conf dir"),
		}
		return
	}

	err = utils.MkdirAll(config.Config.IpsecDirPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "ipsec: Failed to create ipsec conf dir"),
		}
		return
	}

	return
}

func writeConf() (err error) {
	data := fmt.Sprintf("include %s/*.conf", config.Config.IpsecDirPath)

	pth := path.Join(config.Config.IpsecConfPath)
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

	for _, stat := range states {
		confBuf := &bytes.Buffer{}

		for i, link := range stat.Links {
			data := &templateData{
				Id:           fmt.Sprintf("%s-%d", stat.Id, i),
				Left:         state.GetPublicAddress(),
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

		pth := path.Join(config.Config.IpsecDirPath,
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
		config.Config.IpsecSecretsPath, secretsBuf.Bytes(), 0600)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write state secrets"),
		}
		return
	}

	return
}

func deploy(states []*state.State) (err error) {
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

	err = advertise.AdvertisePorts()
	if err != nil {
		return
	}

	err = utils.Exec("", "ipsec", "restart")
	if err != nil {
		return
	}

	err = advertise.AdvertiseRoutes(states)
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

func ReDeploy() {
	deployLock.Lock()
	if deployStates == nil && curStates != nil {
		deployStates = curStates
	}
	deployLock.Unlock()
}

func runDeploy() {
	for {
		if deployStates != nil {
			deployLock.Lock()
			states := deployStates
			deployStates = nil
			if states != nil {
				curStates = states
			}
			deployLock.Unlock()

			if states != nil {
				logrus.WithFields(logrus.Fields{
					"local_address":  state.GetLocalAddress(),
					"public_address": state.GetPublicAddress(),
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
				}
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func init() {
	module := requires.New("ipsec")
	module.After("logger")

	module.Handler = func() {
		go runDeploy()
	}
}
