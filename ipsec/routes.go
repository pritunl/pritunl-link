package ipsec

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"time"
)

var (
	routesPeer         = ""
	routesGateway      = ""
	routesDefaultIface = ""
)

func getDirectStatus(stat *state.State) bool {
	status := state.Status

	if stat == nil {
		return false
	}

	stateStatus, ok := status[stat.Id]
	if !ok {
		return false
	}

	linkStatus, ok := stateStatus["0"]
	if !ok {
		return false
	}

	return linkStatus == "connected"
}

func addDirectRoute(peer, gateway, defaultIface string) (err error) {
	DelDirectRoute()

	if constants.Interrupt {
		return
	}

	utils.ExecSilent("",
		"ip", "route",
		"del", peer,
		"via", gateway,
		"dev", defaultIface,
	)
	err = utils.Exec("",
		"ip", "route",
		"add", peer,
		"via", gateway,
		"dev", defaultIface,
	)
	if err != nil {
		utils.ExecSilent("",
			"ip", "route",
			"del", peer,
			"via", gateway,
			"dev", defaultIface,
		)
		return
	}

	err = utils.Exec("",
		"ip", "route",
		"add", "0.0.0.0/0",
		"dev", DirectIface,
	)
	if err != nil {
		utils.ExecSilent("",
			"ip", "route",
			"del", peer,
			"via", gateway,
			"dev", defaultIface,
		)
		utils.ExecSilent("",
			"ip", "route",
			"del", "0.0.0.0/0",
			"dev", DirectIface,
		)
		return
	}

	return
}

func DelDirectRoute() {
	if routesPeer != "" && routesGateway != "" && routesDefaultIface != "" {
		utils.ExecSilent("",
			"ip", "route",
			"del", routesPeer,
			"via", routesGateway,
			"dev", routesDefaultIface,
		)
	}

	utils.ExecSilent("",
		"ip", "route",
		"del", "0.0.0.0/0",
		"dev", DirectIface,
	)
}

func runRoutes() {
	for {
		time.Sleep(100 * time.Millisecond)
		if constants.Interrupt {
			return
		}

		newRoutesPeer := ""
		ipsecState := state.DirectIpsecState
		if ipsecState != nil && len(ipsecState.Links) == 0 {
			newRoutesPeer = ipsecState.Links[0].Right
		}
		newRoutesGateway := state.GetDefaultGateway()
		newRoutesDefaultIface := state.GetDefaultInterface()
		directStatus := getDirectStatus(ipsecState)

		if newRoutesPeer != "" && directStatus {
			if routesPeer != newRoutesPeer ||
				routesGateway != newRoutesGateway ||
				routesDefaultIface != newRoutesDefaultIface {

				logrus.WithFields(logrus.Fields{
					"peer":          newRoutesPeer,
					"gateway":       newRoutesGateway,
					"default_iface": newRoutesDefaultIface,
				}).Info("ipsec: Adding IPsec routes")

				err := addDirectRoute(
					newRoutesPeer,
					newRoutesGateway,
					newRoutesDefaultIface,
				)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"error": err,
					}).Error("ipsec: Failed to add IPsec routes")

					time.Sleep(3 * time.Second)

					continue
				}

				routesPeer = newRoutesPeer
				routesGateway = newRoutesGateway
				routesDefaultIface = newRoutesDefaultIface
			}
		} else if routesPeer != "" {
			logrus.WithFields(logrus.Fields{
				"peer":          routesPeer,
				"gateway":       routesGateway,
				"default_iface": routesDefaultIface,
			}).Info("ipsec: Removing IPsec routes")

			DelDirectRoute()
		}
	}
}
