package config

import (
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/requires"
	"github.com/pritunl/pritunl-link/utils"
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
	DisablePort bool   `json:"disable_port"`
	Controller  string `json:"controller"`
	Site        string `json:"site"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Interface   string `json:"interface"`
}

type ConfigData struct {
	loaded                     bool       `json:"-"`
	Provider                   string     `json:"provider"`
	PublicAddress              string     `json:"public_address"`
	LocalAddress               string     `json:"local_address"`
	Uris                       []string   `json:"uris"`
	SkipVerify                 bool       `json:"skip_verify"`
	DeleteRoutes               bool       `json:"delete_routes"`
	DisconnectedTimeout        int        `json:"disconnected_timeout"`
	DisableAdvertiseUpdate     bool       `json:"disable_advertise_update"`
	DisableDisconnectedRestart bool       `json:"disable_disconnected_restart"`
	Aws                        AwsData    `json:"aws"`
	Google                     GoogleData `json:"google"`
	Unifi                      UnifiData  `json:"unifi"`
}

func (c *ConfigData) Save() (err error) {
	if !c.loaded {
		err = &errortypes.WriteError{
			errors.New("config: Config file has not been loaded"),
		}
		return
	}

	data, err := json.MarshalIndent(c, "", "\t")
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

	exists, err := utils.Exists(constants.ConfPath)
	if err != nil {
		return
	}

	if !exists {
		data.loaded = true
		Config = data
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

func GetModTime() (mod time.Time, err error) {
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

func init() {
	module := requires.New("config")

	module.Handler = func() {
		utils.ExistsMkdir(constants.VarDir, 0755)

		err := Load()
		if err != nil {
			panic(err)
		}

		exists, err := utils.Exists(constants.ConfPath)
		if err != nil {
			panic(err)
		}

		if !exists {
			err := Save()
			if err != nil {
				panic(err)
			}
		}
	}
}
