package watch

import (
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/state"
	"time"
)

func watchUnknown() {
	for {
		time.Sleep(10 * time.Second)

		states := ipsec.GetStates()

		unknownIds, err := state.Unknown(states)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("state: Unknown check error")
			continue
		}

		if unknownIds != nil && len(unknownIds) > 0 {
			unknownIdsSet := set.NewSet()

			for _, linkId := range unknownIds {
				unknownIdsSet.Add(linkId)
			}

			time.Sleep(5 * time.Second)

			states2 := ipsec.GetStates()

			unknownIds2, err := state.Unknown(states2)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Info("state: Unknown check error")
				continue
			}

			for _, linkId := range unknownIds2 {
				if unknownIdsSet.Contains(linkId) {
					logrus.WithFields(logrus.Fields{
						"link_id": linkId,
					}).Info("state: Stopping unknown link")
					go ipsec.Shutdown(linkId)
				}
			}
		}
	}
}
