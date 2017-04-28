package ipsec

import (
	"bytes"
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"io/ioutil"
	"os"
	"path"
	"strings"
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
	err = os.RemoveAll(config.Config.IpsecDirPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "ipsec: Failed to remove ipsec conf dir"),
		}
		return
	}

	err = os.MkdirAll(config.Config.IpsecDirPath, 0755)
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
	err = ioutil.WriteFile(pth, []byte(data), 0600)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write conf"),
		}
		return
	}

	return
}

func writeTemplates() (err error) {
	states := state.States
	secretsBuf := &bytes.Buffer{}

	for _, stat := range states {
		confBuf := &bytes.Buffer{}

		for i, link := range stat.Links {
			data := &templateData{
				Id:           fmt.Sprintf("%s-%d", stat.Id, i),
				Left:         config.Config.PublicAddress,
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
		err = ioutil.WriteFile(pth, confBuf.Bytes(), 0600)
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

func Deploy() (err error) {
	err = clearDir()
	if err != nil {
		return
	}

	err = writeConf()
	if err != nil {
		return
	}

	err = writeTemplates()
	if err != nil {
		return
	}

	err = utils.Exec("", "service", "ipsec", "restart")
	if err != nil {
		return
	}

	return
}
