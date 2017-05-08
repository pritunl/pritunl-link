package config

import (
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/requires"
	"io/ioutil"
	"os"
)

var (
	confPath = "/etc/pritunl-link.json"
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

type ConfigData struct {
	path             string      `json:"-"`
	loaded           bool        `json:"-"`
	Provider         string      `json:"provider"`
	PublicAddress    string      `json:"public_address"`
	Uris             []string    `json:"uris"`
	SkipVerify       bool        `json:"skip_verify"`
	IpsecConfPath    string      `json:"ipsec_conf_path"`
	IpsecSecretsPath string      `json:"ipsec_secrets_path"`
	IpsecDirPath     string      `json:"ipsec_dir_path"`
	Aws              *AwsData    `json:"aws"`
	Google           *GoogleData `json:"google"`
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

	err = ioutil.WriteFile(c.path, data, 0600)
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

	data.path = confPath

	_, err = os.Stat(data.path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			data.loaded = true
		} else {
			err = &errortypes.ReadError{
				errors.Wrap(err, "config: File stat error"),
			}
		}
		return
	}

	file, err := ioutil.ReadFile(data.path)
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
	}
}
