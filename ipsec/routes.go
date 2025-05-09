package ipsec

import (
	"fmt"
	"time"

	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/status"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
)

var (
	routesPeer         = ""
	routesGateway      = ""
	routesDefaultIface = ""
)

func getDirectStatus(stat *state.State) (directStatus bool, err error) {
	if stat == nil {
		return
	}

	if len(stat.Links) == 0 {
		return
	}

	var stats status.Status
	if stat.Protocol == "wg" {
		linkId := fmt.Sprintf("%s-%s-%s", stat.Id, stat.Links[0].Id, stat.Hash)

		wgKeyMap := map[string]string{}
		for _, lnk := range stat.Links {
			if stat.Protocol == "wg" {
				wgKeyMap[lnk.WgPublicKey] = fmt.Sprintf(
					"%s-%s", stat.Id, lnk.Id, stat.Hash)
			}
		}

		stats, err = status.GetWg(wgKeyMap)
		if err != nil {
			return
		}

		linkStatus, ok := stats[linkId]
		if !ok {
			return
		}

		directStatus = linkStatus == "connected"
	} else if stat.Protocol == "" || stat.Protocol == "ipsec" {
		linkId := fmt.Sprintf("%s-0-%s", stat.Id, stat.Links[0].Hash)

		stats, err = status.Get()
		if err != nil {
			return
		}

		linkStatus, ok := stats[linkId]
		if !ok {
			return
		}

		directStatus = linkStatus == "connected"
	}

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
