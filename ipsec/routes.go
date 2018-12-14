package ipsec

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/status"
	"github.com/pritunl/pritunl-link/utils"
	"time"
)

var (
	routesPeer         = ""
	routesGateway      = ""
	routesDefaultIface = ""
)

func getDirectStatus(stat *state.State) (directStatus bool, err error) {
	stats, err := status.Get()
	if err != nil {
		return
	}

	if stat == nil {
		return
	}

	if stat.Links == nil || len(stat.Links) == 0 {
		return
	}

	linkId := fmt.Sprintf("%s-0-%s", stat.Id, stat.Links[0].Hash)

	linkStatus, ok := stats[linkId]
	if !ok {
		return
	}

	directStatus = linkStatus == "connected"

	return
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

	routesPeer = ""
	routesGateway = ""
	routesDefaultIface = ""
}

func runRoutes() {
	for {
		time.Sleep(300 * time.Millisecond)
		if constants.Interrupt {
			return
		}

		newRoutesPeer := ""
		directStatus := false
		ipsecState := state.DirectIpsecState
		if ipsecState != nil && len(ipsecState.Links) > 0 {
			newRoutesPeer = ipsecState.Links[0].Right

			dirctStatus, err := getDirectStatus(ipsecState)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Error("sync: Failed to get status")

				time.Sleep(3 * time.Second)

				continue
			}

			directStatus = dirctStatus
		}
		newRoutesGateway := state.GetDefaultGateway()
		newRoutesDefaultIface := state.GetDefaultInterface()

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
