package routes

import (
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"io/ioutil"
	"os"
)

type CurrentRoutes struct {
	Aws    map[string]*AwsRoute    `json:"aws"`
	Google map[string]*GoogleRoute `json:"google"`
}

func (c *CurrentRoutes) Commit() (err error) {
	data, err := json.Marshal(c)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "advertise: Failed to prase routes"),
		}
		return
	}

	err = ioutil.WriteFile(constants.CurRoutesPath, data, 0644)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "advertise: Failed to write routes"),
		}
		return
	}

	return
}

func GetCurrent() (routes *CurrentRoutes, err error) {
	routes = &CurrentRoutes{}

	if _, e := os.Stat(constants.CurRoutesPath); os.IsNotExist(e) {
		return
	}

	data, err := ioutil.ReadFile(constants.CurRoutesPath)
	if err != nil {
		err = errortypes.ReadError{
			errors.Wrap(err, "advertise: Failed to read routes"),
		}
		return
	}

	err = json.Unmarshal(data, routes)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "advertise: Failed to prase routes"),
		}
		return
	}

	return
}
