package main

import (
	"flag"
	"github.com/pritunl/pritunl-link/cmd"
	"github.com/pritunl/pritunl-link/logger"
	"github.com/pritunl/pritunl-link/requires"
)

func main() {
	flag.Parse()
	logger.Init()

	requires.Init()

	switch flag.Arg(0) {
	case "start":
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		break
	case "add":
		err := cmd.Add(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "remove":
		err := cmd.Remove(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "clear":
		err := cmd.Clear()
		if err != nil {
			panic(err)
		}
		break
	case "local-address":
		err := cmd.LocalAddress(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "public-address":
		err := cmd.PublicAddress(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "verify-on":
		err := cmd.VerifyOn()
		if err != nil {
			panic(err)
		}
		break
	case "verify-off":
		err := cmd.VerifyOff()
		if err != nil {
			panic(err)
		}
		break
	case "disconnected-timeout-on":
		err := cmd.DisconnectedTimeoutOn()
		if err != nil {
			panic(err)
		}
		break
	case "disconnected-timeout-off":
		err := cmd.DisconnectedTimeoutOff()
		if err != nil {
			panic(err)
		}
		break
	case "provider":
		err := cmd.Provider(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "unifi-username":
		err := cmd.UnifiUsername(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "unifi-password":
		err := cmd.UnifiPassword(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	case "unifi-controller":
		err := cmd.UnifiController(flag.Arg(1))
		if err != nil {
			panic(err)
		}
		break
	}
}
