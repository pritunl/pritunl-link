package advertise

import (
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
)

func Routes(states []*state.State) (err error) {
	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	availableNetworks := []string{}
	allNetworks := []string{}

	for _, stat := range states {
		if stat.Type == state.DirectClient ||
			stat.Type == state.DirectServer {

			continue
		}

		for _, link := range stat.Links {
			for _, network := range link.RightSubnets {
				if !stat.Cached {
					availableNetworks = append(availableNetworks, network)
				}

				allNetworks = append(allNetworks, network)
			}
		}
	}

	sort.Strings(availableNetworks)
	sort.Strings(allNetworks)

	curRoutes, err := routes.GetDiff(allNetworks)
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

	if curRoutes.Oracle != nil {
		for _, route := range curRoutes.Oracle {
			err = OracleDeleteRoute(route)
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

	if curRoutes.Azure != nil {
		for _, route := range curRoutes.Azure {
			err = AzureDeleteRoute(route)
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

	if curRoutes.Edge != nil {
		for _, route := range curRoutes.Edge {
			err = EdgeDeleteRoute(route)
			if err != nil {
				return
			}
		}
	}

	if curRoutes.Pritunl != nil {
		for _, route := range curRoutes.Pritunl {
			err = PritunlDeleteRoute(route)
			if err != nil {
				return
			}
		}
	}

	for _, network := range availableNetworks {
		switch config.Config.Provider {
		case "aws":
			err = AwsAddRoute(network)
			if err != nil {
				return
			}

			break
		case "azure":
			err = AzureAddRoute(network)
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
		case "oracle":
			err = OracleAddRoute(network)
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
		case "edge":
			err = EdgeAddRoute(network)
			if err != nil {
				return
			}

			break
		case "pritunl":
			err = PritunlAddRoute(network)
			if err != nil {
				return
			}

			break
		}
	}

	err = os.MkdirAll(constants.VarDir, 0755)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "advertise: Failed to create var directory"),
		}
		return
	}

	data := strings.Join(allNetworks, "\n")
	if data != "" {
		data = data + "\n"
	}
	err = ioutil.WriteFile(constants.RoutesPath, []byte(data), 0644)
	if err != nil {
		err = &errortypes.WriteError{
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
	case "edge":
		if !config.Config.Unifi.DisablePort {
			err = EdgeAddPorts()
			if err != nil {
				return
			}
		}

		break
	}

	return
}
