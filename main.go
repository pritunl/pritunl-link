package main

import (
	"flag"
	"github.com/pritunl/pritunl-link/cmd"
	"github.com/pritunl/pritunl-link/logger"
	"github.com/pritunl/pritunl-link/requires"
)

func main() {
	flag.Parse()

	requires.Init()
	logger.Init()

	switch flag.Arg(0) {
	case "start":
		cmd.Start()
	}
}
