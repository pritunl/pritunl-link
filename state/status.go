package state

import (
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/status"
	"time"
)

var (
	offlineTime time.Time
)

func Update(total int) (timeout bool, err error) {
	stats, connected, err := status.Get()
	if err != nil {
		return
	}

	Status = stats

	if connected < total {
		if !offlineTime.IsZero() {
			disconnectedTimeout := constants.DefaultDiconnectedTimeout

			configTimeout := config.Config.DisconnectedTimeout
			if configTimeout != 0 {
				disconnectedTimeout = time.Duration(
					configTimeout) * time.Second
			}

			if !config.Config.DisableDisconnectedRestart {
				if time.Since(offlineTime) > disconnectedTimeout {
					timeout = true
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
