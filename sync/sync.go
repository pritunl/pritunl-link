package sync

import (
	"github.com/pritunl/pritunl-link/state"
	"time"
)

func Init() {
	state.SyncStates()
	for {
		time.Sleep(1 * time.Second)
		state.SyncStates()
	}
}
