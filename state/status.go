package state

import (
	"fmt"
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
	names := set.NewSet()
	for _, stat := range states {
		for i := range stat.Links {
			names.Add(fmt.Sprintf("%s-%d", stat.Id, i))
		}
	}

	stats, _, err := status.Get()
	if err != nil {
		return
	}

	unknown := set.NewSet()
	unknownIds = []string{}

	for stateId, conns := range stats {
		for connId, _ := range conns {
			id := fmt.Sprintf("%s-%s", stateId, connId)

			if !names.Contains(id) && len(stateId) == 24 &&
				!unknown.Contains(id) {

				unknown.Add(id)
				unknownIds = append(unknownIds, id)
			}
		}
	}

	return
}

func Update(states []*State) (hasConnected bool,
	resetLinks []string, err error) {

	resetLinks = []string{}

	names := set.NewSet()
	for _, stat := range states {
		for i := range stat.Links {
			names.Add(fmt.Sprintf("%s-%d", stat.Id, i))
		}
	}

	stats, _, err := status.Get()
	if err != nil {
		return
	}

	Status = stats

	for stateId, conns := range stats {
		for connId, connStatus := range conns {
			id := fmt.Sprintf("%s-%s", stateId, connId)

			if connStatus == "connected" {
				if names.Contains(id) {
					hasConnected = true
					names.Remove(id)
				}
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
