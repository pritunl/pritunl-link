package advertise

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
	"io/ioutil"
	"os"
	"strings"
)

func Routes(states []*state.State) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	networks := []string{}

	for _, stat := range states {
		for _, link := range stat.Links {
			for _, network := range link.RightSubnets {
				networks = append(networks, network)
			}
		}
	}

	curRoutes, err := routes.GetDiff(networks)
	if err != nil {
		return
	}

	if curRoutes.Google != nil {
		for _, route := range curRoutes.Google {
			err = GoogleDeleteRoute(route)
			if err != nil {
				return
			}
		}
	}

	if curRoutes.Aws != nil {
		for _, route := range curRoutes.Aws {
			err = AwsDeleteRoute(route)
			if err != nil {
				return
			}
		}
	}

	if curRoutes.Unifi != nil {
		for _, route := range curRoutes.Unifi {
			err = UnifiDeleteRoute(route)
			if err != nil {
				return
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
		case "unifi":
			err = UnifiAddRoute(network)
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

func Ports(states []*state.State) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	hasLinks := false
	for _, ste := range states {
		if ste.Links != nil && len(ste.Links) != 0 {
			hasLinks = true
		}
	}

	if !hasLinks {
		return
	}

	switch config.Config.Provider {
	case "unifi":
		if !config.Config.Unifi.DisablePort {
			err = UnifiAddPorts()
			if err != nil {
				return
			}
		}

		break
	}

	return
}
