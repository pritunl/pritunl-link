package advertise

import (
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/state"
)

func AdvertiseRoutes() (err error) {
	states := state.States

	for _, stat := range states {
		for _, link := range stat.Links {
			for _, network := range link.RightSubnets {
				if config.Config.Provider == "aws" {
					err = AwsAddRoute(network)
					if err != nil {
						return
					}
				} else if config.Config.Provider == "google" {
					err = GoogleAddRoute(network)
					if err != nil {
						return
					}
				}
			}
		}
	}

	return
}
