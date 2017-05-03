package advertise

import (
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"io/ioutil"
	"os"
)

type AwsRoute struct {
	Network     string `json:"network"`
	InterfaceId string `json:"interface_id"`
	InstanceId  string `json:"instance_id"`
}

func (r *AwsRoute) Add() (err error) {
	routes, err := getCurrentRoutes()
	if err != nil {
		return
	}

	if routes.Aws == nil {
		routes.Aws = map[string]*AwsRoute{}
	}

	routes.Aws[r.Network] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

type currentRoutes struct {
	Aws map[string]*AwsRoute `json:"aws"`
}

func (c *currentRoutes) Commit() (err error) {
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

func getCurrentRoutes() (routes *currentRoutes, err error) {
	routes = &currentRoutes{}

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
