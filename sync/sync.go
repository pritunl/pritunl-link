package sync

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/ipsec"
	"github.com/pritunl/pritunl-link/state"
	"io"
	"time"
)

func SyncStates() {
	states := state.GetStates()
	hsh := md5.New()

	for _, stat := range states {
		io.WriteString(hsh, stat.Hash)
	}

	newHash := hex.EncodeToString(hsh.Sum(nil))

	if newHash != state.Hash {
		logrus.Info("state: Deploying state")

		state.States = states

		err := ipsec.Deploy()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Info("state: Failed to deploy state")
			time.Sleep(1 * time.Second)
			return
		}

		state.Hash = newHash
	}
}

func Init() {
	SyncStates()
	for {
		time.Sleep(1 * time.Second)
		SyncStates()
		status, _ := ipsec.GetStatus()
		fmt.Println(status)
	}
}
