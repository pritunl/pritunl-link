package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pritunl/pritunl-link/clean"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/interlink"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/sync"
	"github.com/pritunl/pritunl-link/watch"
	"github.com/sirupsen/logrus"
)

func Start() (err error) {
	logrus.WithFields(logrus.Fields{
		"version": constants.Version,
	}).Info("cmd.start: Starting link")

	err = state.Init()
	if err != nil {
		return
	}

	sync.Init()
	watch.Init()

	err = interlink.Init()
	if err != nil {
		return
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	constants.Interrupt = true

	clean.CleanUp()

	time.Sleep(1010 * time.Millisecond)

	clean.CleanUp()

	return
}
