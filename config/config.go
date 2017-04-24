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

type ConfigData struct {
	path       string   `json:"-"`
	loaded     bool     `json:"-"`
	PublicAddr string   `json:"public_addr"`
	Uris       []string `json:"uris"`
}

func (c *ConfigData) Load(path string) (err error) {
	c.path = path

	_, err = os.Stat(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			c.loaded = true
		} else {
			err = &errortypes.ReadError{
				errors.Wrap(err, "config: File stat error"),
			}
		}
		return
	}

	file, err := ioutil.ReadFile(c.path)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "config: File read error"),
		}
		return
	}

	err = json.Unmarshal(file, Config)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "config: File unmarshal error"),
		}
		return
	}

	if c.Uris == nil {
		c.Uris = []string{}
	}

	c.loaded = true

	return
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
	err = Config.Load(confPath)
	if err != nil {
		return
	}

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
