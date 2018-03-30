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
	offlineTime time.Time
)

func Update(names set.Set) (resetLinks []string, err error) {
	resetLinks = []string{}

	stats, _, err := status.Get()
	if err != nil {
		return
	}

	Status = stats

	unknown := set.NewSet()
	for stateId, conns := range stats {
		for connId, connStatus := range conns {
			id := fmt.Sprintf("%s-%s", stateId, connId)

			if connStatus == "connected" {
				if names.Contains(id) {
					names.Remove(id)
				} else {
					unknown.Add(id)
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
					for nameInf := range names.Iter() {
						resetLinks = append(resetLinks, nameInf.(string))
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
