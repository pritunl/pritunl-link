package advertise

import (
	"context"
	"net"

	"github.com/dropbox/godropbox/errors"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/state"
)

func HetznerDeleteRoute(route *routes.HetznerRoute) (err error) {
	client := hcloud.NewClient(hcloud.WithToken(config.Config.Hetzner.Token))
	network, err := HetznerGetNetwork(*client)
	if err != nil {
		return
	}

	for _, rte := range network.Routes {
		if rte.Destination.String() == route.Destination &&
			rte.Gateway.String() == route.Gateway {

			_, _, err = client.Network.DeleteRoute(
				context.Background(),
				network,
				hcloud.NetworkDeleteRouteOpts{
					Route: rte,
				},
			)
			if err != nil {
				err = &errortypes.RequestError{
					errors.Wrap(err, "hetzner: Failed to delete route"),
				}
				return
			}
		}
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}

func HetznerGetNetwork(client hcloud.Client) (
	network *hcloud.Network, err error) {

	network, _, err = client.Network.GetByID(
		context.Background(), config.Config.Hetzner.NetworkId)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "hetzner: Failed to get network"),
		}
		return
	}

	if network == nil {
		err = &errortypes.RequestError{
			errors.New("hetzner: Network not found"),
		}
		return
	}

	return
}

func HetznerAddRoute(destination string) (err error) {
	gateway := state.GetLocalAddress()

	client := hcloud.NewClient(hcloud.WithToken(config.Config.Hetzner.Token))

	network, err := HetznerGetNetwork(*client)
	if err != nil {
		return
	}

	existingRoute := false
	for _, route := range network.Routes {
		if route.Destination.String() == destination {
			if route.Gateway.String() != gateway {
				_, _, err = client.Network.DeleteRoute(
					context.Background(),
					network,
					hcloud.NetworkDeleteRouteOpts{
						Route: route,
					},
				)
				if err != nil {
					err = &errortypes.RequestError{
						errors.New("hetzner: Failed to delete route"),
					}
					return
				}
			} else {
				existingRoute = true
			}
		}
	}

	if existingRoute == false {
		_, destinationIpNet, _ := net.ParseCIDR(destination)
		gatewayIp := net.ParseIP(gateway)

		route := hcloud.NetworkRoute{
			Destination: destinationIpNet,
			Gateway:     gatewayIp,
		}

		_, _, err = client.Network.AddRoute(
			context.Background(),
			network,
			hcloud.NetworkAddRouteOpts{
				Route: route,
			},
		)
		if err != nil {
			err = &errortypes.RequestError{
				errors.New("hetzner: Failed to create route"),
			}
			return
		}
	}

	route := &routes.HetznerRoute{
		Destination: destination,
		Gateway:     gateway,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}
