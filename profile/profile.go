package profile

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"io/ioutil"
	"strings"
)

var (
	ConfDir      string
	Host         string
	Token        string
	Secret       string
	Username     string
	NetworkLinks []string
)

type Profile struct {
	UserId         string   `json:"user_id"`
	OrganizationId string   `json:"organization_id"`
	ServerId       string   `json:"server_id"`
	SyncHash       string   `json:"sync_hash"`
	SyncToken      string   `json:"sync_token"`
	SyncSecret     string   `json:"sync_secret"`
	SyncHosts      []string `json:"sync_hosts"`
	Conf           string   `json:"conf"`
}

func (p *Profile) update(data string) (err error) {
	keyData := ""

	if strings.Contains(p.Conf, "key-direction") && strings.Contains(
		data, "key-direction") {

		keyData += "key-direction 1\n"
	}

	sIndex := strings.Index(p.Conf, "<tls-auth>")
	eIndex := strings.Index(p.Conf, "</tls-auth>")
	if sIndex != 0 && eIndex != 0 {
		keyData += p.Conf[sIndex:eIndex+11] + "\n"
	}

	sIndex = strings.Index(p.Conf, "<cert>")
	eIndex = strings.Index(p.Conf, "</cert>")
	if sIndex != 0 && eIndex != 0 {
		keyData += p.Conf[sIndex:eIndex+7] + "\n"
	}

	sIndex = strings.Index(p.Conf, "<key>")
	eIndex = strings.Index(p.Conf, "</key>")
	if sIndex != 0 && eIndex != 0 {
		keyData += p.Conf[sIndex:eIndex+6] + "\n"
	}

	err = p.Parse(data + keyData)
	if err != nil {
		return
	}

	return
}

func (p *Profile) Parse(data string) (err error) {
	lines := strings.Split(data, "\n")
	jsonData := ""
	conf := ""

	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			jsonData += strings.TrimSpace(line[1:])
		} else {
			conf = strings.Join(lines[i:], "\n")
			break
		}
	}

	err = json.Unmarshal([]byte(jsonData), p)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "profile: Failed to parse json data"),
		}
		return
	}

	p.Conf = conf

	return
}

func (p *Profile) Sync() (err error) {
	path := fmt.Sprintf("/key/%s/%s/%s/%s",
		p.OrganizationId,
		p.UserId,
		p.ServerId,
		p.SyncHash,
	)

	for i, host := range p.SyncHosts {
		resp, e := AuthReq(p.SyncToken, p.SyncSecret, sha512.New, host,
			"GET", path, nil)
		if e != nil {
			err = e
			return
		}

		switch resp.StatusCode {
		case 401:
			logrus.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Error("profile: Failed to sync profile, auth failed")
			return
		case 404:
			logrus.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Error("profile: Failed to sync profile, user not found")
			return
		case 480:
			logrus.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Error("profile: Failed to sync profile, no subscription")
			return
		case 200:
			body, e := ioutil.ReadAll(resp.Body)
			if e != nil {
				err = e
				return
			}
			bodyStr := string(body)

			if bodyStr != "" {
				err = p.update(bodyStr)
				if err != nil {
					return
				}
			}
		default:
			if i == len(p.SyncHosts)-1 {
				logrus.WithFields(logrus.Fields{
					"status_code": resp.StatusCode,
				}).Error("profile: Failed to sync profile, unknown error")
				return
			}
		}
	}

	return
}
