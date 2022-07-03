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

func HcloudDeleteRoute(route *routes.HcloudRoute) (err error) {
	client := hcloud.NewClient(hcloud.WithToken(config.Config.Hcloud.Token))
	network, err := HcloudGetNetwork(*client)
	if err != nil {
		return
	}

	for _, route := range network.Routes {
		if route.Destination.String() == route.Destination.String() && route.Gateway.String() != route.Gateway.String() {
			_, _, err = client.Network.DeleteRoute(context.Background(), network, hcloud.NetworkDeleteRouteOpts{Route: route})
			if err != nil {
				err = &errortypes.RequestError{
					errors.Wrap(err, "hcloud: Failed to delete route"),
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

func HcloudGetNetwork(client hcloud.Client) (network *hcloud.Network, err error) {
	network, _, err = client.Network.GetByID(context.Background(), config.Config.Hcloud.NetworkId)

	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "hcloud: request failed"),
		}
		return
	}

	if network == nil {
		err = &errortypes.RequestError{
			errors.New("hcloud: network not found"),
		}
		return
	}

	return network, nil
}

func HcloudAddRoute(destination string) (err error) {
	gateway := state.GetLocalAddress()

	client := hcloud.NewClient(hcloud.WithToken(config.Config.Hcloud.Token))

	network, err := HcloudGetNetwork(*client)
	if err != nil {
		return
	}

	existingRoute := false
	for _, route := range network.Routes {
		if route.Destination.String() == destination {
			if route.Gateway.String() != gateway {
				// wrong gateway delete route
				_, _, err = client.Network.DeleteRoute(context.Background(), network, hcloud.NetworkDeleteRouteOpts{Route: route})
				if err != nil {
					err = &errortypes.RequestError{
						errors.New("hcloud: Failed to delete route"),
					}
					return
				}
			} else {
				existingRoute = true
			}
		}
	}

	if existingRoute == false {
		// add new route
		_, destinationIpNet, _ := net.ParseCIDR(destination)
		gatewayIp := net.ParseIP(gateway)
		route := hcloud.NetworkRoute{Destination: destinationIpNet, Gateway: gatewayIp}
		_, _, err = client.Network.AddRoute(context.Background(), network, hcloud.NetworkAddRouteOpts{Route: route})
		if err != nil {
			err = &errortypes.RequestError{
				errors.New("hcloud: Failed to create route"),
			}
			return
		}
	}

	route := &routes.HcloudRoute{
		Destination: destination,
		Gateway:     gateway,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}
