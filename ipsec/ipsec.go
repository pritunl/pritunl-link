package ipsec

import (
	"bytes"
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/state"
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
	files, err := ioutil.ReadDir(config.Config.IpsecDirPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "ipsec: Failed to read ipsec dir"),
		}
		return
	}

	for _, file := range files {
		pth := path.Join(config.Config.IpsecDirPath, file.Name())
		err = os.RemoveAll(pth)
		if err != nil {
			err = &errortypes.WriteError{
				errors.Wrap(err, "ipsec: Failed to remove ipsec file"),
			}
			return
		}
	}

	return
}

func writeConf() (err error) {
	pth := path.Join(config.Config.IpsecConfPath)
	err = ioutil.WriteFile(pth, []byte(conf), 0600)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write conf"),
		}
		return
	}

	pth = path.Join(config.Config.IpsecSecretsPath)
	err = ioutil.WriteFile(pth, []byte(secrets), 0600)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "ipsec: Failed to write secrets"),
		}
		return
	}

	return
}

func writeTemplates() (err error) {
	states := state.States

	for _, stat := range states {
		confBuf := &bytes.Buffer{}
		secretsBuf := &bytes.Buffer{}

		for i, link := range stat.Links {
			data := &templateData{
				Id:           fmt.Sprintf("%s-%d", stat.Id, i),
				Left:         config.Config.PublicAddress,
				LeftSubnets:  strings.Join(link.LeftSubnets, " "),
				Right:        link.Right,
				RightSubnets: strings.Join(link.RightSubnets, " "),
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

		pth = path.Join(config.Config.IpsecDirPath,
			fmt.Sprintf("%s.secrets", stat.Id))
		err = ioutil.WriteFile(pth, secretsBuf.Bytes(), 0600)
		if err != nil {
			err = errortypes.WriteError{
				errors.Wrap(err, "ipsec: Failed to write state secrets"),
			}
			return
		}
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

	return
}
