package logger

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/colorize"
)

var (
	blueArrow    = colorize.ColorString("▶", colorize.BlueBold, colorize.None)
	whiteDiamond = colorize.ColorString("◆", colorize.WhiteBold, colorize.None)
)

type formatter struct{}

func (f *formatter) Format(entry *logrus.Entry) (output []byte, err error) {
	msg := fmt.Sprintf("%s %s %s", formatLevel(entry.Level), blueArrow,
		entry.Message)

	var errStr string
	for key, val := range entry.Data {
		if key == "error" {
			errStr = fmt.Sprintf("%s", val)
			continue
		}

		msg += fmt.Sprintf(" %s %s=%v", whiteDiamond,
			colorize.ColorString(key, colorize.CyanBold, colorize.None),
			colorize.ColorString(fmt.Sprintf("%#v", val),
				colorize.GreenBold, colorize.None))
	}

	if errStr != "" {
		msg += "\n" + colorize.ColorString(errStr, colorize.Red, colorize.None)
	}

	if string(msg[len(msg)-1]) != "\n" {
		msg += "\n"
	}

	output = []byte(msg)

	return
}
