package sync

import (
	"crypto/md5"
	"encoding/hex"
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
		state.Hash = newHash
		state.States = states
	}
}

func Init() {
	SyncStates()
	for {
		time.Sleep(1 * time.Second)
		SyncStates()
	}
}
