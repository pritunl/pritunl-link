package state

import (
	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/status"
	"time"
)

var (
	offlineTime   time.Time
	lastReconnect = time.Now()
)

func Unknown(states []*State) (unknownIds []string, err error) {
	connIds := set.NewSet()
	for _, stat := range states {
		for i, lnk := range stat.Links {
			connIds.Add(GetLinkId(stat.Id, i, lnk.Hash))

			if lnk.Static && (len(lnk.LeftSubnets) > 1 ||
				len(lnk.RightSubnets) > 1) {

				for x := range lnk.LeftSubnets {
					for y := range lnk.RightSubnets {
						connIds.Add(GetLinkIds(
							stat.Id, i, x, y, lnk.Hash))
					}
				}
			}
		}
	}

	curConnIds, err := status.GetIds()
	if err != nil {
		return
	}

	unknown := set.NewSet()
	unknownIds = []string{}

	for _, connId := range curConnIds {
		if !connIds.Contains(connId) && !unknown.Contains(connId) {
			unknown.Add(connId)
			unknownIds = append(unknownIds, connId)
		}
	}

	return
}

func Update(states []*State) (hasConnected bool,
	resetLinks []string, err error) {

	resetLinks = []string{}

	names := set.NewSet()
	for _, stat := range states {
		for i, lnk := range stat.Links {
			names.Add(GetLinkId(stat.Id, i, lnk.Hash))
		}
	}

	stats, err := status.Get()
	if err != nil {
		return
	}

	Status = stats

	for connId, connStatus := range stats {
		if connStatus == "connected" {
			if names.Contains(connId) {
				hasConnected = true
				names.Remove(connId)
			}
		}
	}

	if names.Len() > 0 {
		if !offlineTime.IsZero() {
			disconnectedTimeout := constants.DefaultDiconnectedTimeout

			configTimeout := config.Config.DisconnectedTimeout
			if configTimeout != 0 {
				disconnectedTimeout = time.Duration(
					configTimeout) * time.Second
			}

			if !config.Config.DisableDisconnectedRestart {
				if time.Since(offlineTime) > disconnectedTimeout {
					if time.Since(lastReconnect) >
						constants.DiconnectedTimeoutBackoff {

						for nameInf := range names.Iter() {
							resetLinks = append(resetLinks, nameInf.(string))
						}
						lastReconnect = time.Now()
					}
					offlineTime = time.Time{}
				}
			} else {
				offlineTime = time.Time{}
			}
		} else {
			offlineTime = time.Now()
		}
	} else {
		offlineTime = time.Time{}
	}

	return
}
