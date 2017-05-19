package config

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/requires"
	"io/ioutil"
	"os"
	"time"
)

var Config = &ConfigData{}

type AwsData struct {
	Region      string `json:"region"`
	VpcId       string `json:"vpc_id"`
	InstanceId  string `json:"instance_id"`
	InterfaceId string `json:"interface_id"`
}

type GoogleData struct {
	Project  string `json:"project"`
	Network  string `json:"network"`
	Instance string `json:"instance"`
}

type UnifiData struct {
	Controller string `json:"controller"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Interface  string `json:"interface"`
}

type ConfigData struct {
	loaded           bool       `json:"-"`
	Provider         string     `json:"provider"`
	PublicAddress    string     `json:"public_address"`
	Uris             []string   `json:"uris"`
	SkipVerify       bool       `json:"skip_verify"`
	IpsecConfPath    string     `json:"ipsec_conf_path"`
	IpsecSecretsPath string     `json:"ipsec_secrets_path"`
	IpsecDirPath     string     `json:"ipsec_dir_path"`
	Aws              AwsData    `json:"aws"`
	Google           GoogleData `json:"google"`
	Unifi            UnifiData  `json:"unifi"`
}

func (c *ConfigData) Save() (err error) {
	if !c.loaded {
		err = &errortypes.WriteError{
			errors.New("config: Config file has not been loaded"),
		}
		return
	}

	data, err := json.Marshal(c)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "config: File marshal error"),
		}
		return
	}

	err = ioutil.WriteFile(constants.ConfPath, data, 0600)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "config: File write error"),
		}
		return
	}

	return
}

func Load() (err error) {
	data := &ConfigData{}

	_, err = os.Stat(constants.ConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			data.loaded = true
			Config = data
		} else {
			err = &errortypes.ReadError{
				errors.Wrap(err, "config: File stat error"),
			}
		}
		return
	}

	file, err := ioutil.ReadFile(constants.ConfPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "config: File read error"),
		}
		return
	}

	err = json.Unmarshal(file, data)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "config: File unmarshal error"),
		}
		return
	}

	if data.Uris == nil {
		data.Uris = []string{}
	}

	if data.IpsecConfPath == "" {
		data.IpsecConfPath = "/etc/ipsec.conf"
	}

	if data.IpsecSecretsPath == "" {
		data.IpsecSecretsPath = "/etc/ipsec.secrets"
	}

	if data.IpsecDirPath == "" {
		data.IpsecDirPath = "/etc/ipsec.pritunl"
	}

	data.loaded = true

	Config = data

	return
}

func Save() (err error) {
	err = Config.Save()
	if err != nil {
		return
	}

	return
}

func getModTime() (mod time.Time, err error) {
	stat, err := os.Stat(constants.ConfPath)
	if err != nil {
		err = errortypes.ReadError{
			errors.Wrap(err, "config: Failed to stat conf file"),
		}
		return
	}

	mod = stat.ModTime()

	return
}

func watch() {
	curMod, _ := getModTime()

	for {
		time.Sleep(500 * time.Millisecond)

		mod, err := getModTime()
		if err != nil {
			continue
		}

		if mod != curMod {
			err = Load()
			if err != nil {
				continue
			}

			logrus.Info("Reloaded config")

			curMod = mod
		}
	}
}

func init() {
	module := requires.New("config")

	module.Handler = func() {
		err := Load()
		if err != nil {
			panic(err)
		}

		err = Save()
		if err != nil {
			panic(err)
		}

		go watch()
	}
}
