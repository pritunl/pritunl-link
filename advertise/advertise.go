package advertise

import (
	"github.com/pritunl/pritunl-link/cloud"
	"github.com/pritunl/pritunl-link/state"
)

func AdvertiseRoutes() (err error) {
	states := state.States

	for _, stat := range states {
		for _, link := range stat.Links {
			for _, network := range link.RightSubnets {
				err = cloud.AwsAddRoute(network)
				if err != nil {
					return
				}
			}
		}
	}

	return
}
