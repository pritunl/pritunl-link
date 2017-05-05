package advertise

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/state"
	"io/ioutil"
	"os"
	"strings"
)

func AdvertiseRoutes() (err error) {
	states := state.States
	networks := []string{}

	for _, stat := range states {
		for _, link := range stat.Links {
			for _, network := range link.RightSubnets {
				networks = append(networks, network)
			}
		}
	}

	for _, network := range networks {
		switch config.Config.Provider {
		case "aws":
			err = AwsAddRoute(network)
			if err != nil {
				return
			}

			break
		case "google":
			err = GoogleAddRoute(network)
			if err != nil {
				return
			}

			break
		}
	}

	err = os.MkdirAll(constants.VarDir, 0755)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "advertise: Failed to create var directory"),
		}
		return
	}

	data := strings.Join(networks, "\n")
	err = ioutil.WriteFile(constants.RoutesPath, []byte(data), 0644)
	if err != nil {
		err = errortypes.WriteError{
			errors.Wrap(err, "advertise: Failed to write routes"),
		}
		return
	}

	return
}
