// Logger with output to stderr and papertrail.
package logger

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/colorize"
	"github.com/pritunl/pritunl-link/requires"
	"os"
)

var (
	buffer  = make(chan *logrus.Entry, 32)
	senders = []sender{}
)

func formatLevel(lvl logrus.Level) (str string) {
	var colorBg colorize.Color

	switch lvl {
	case logrus.InfoLevel:
		colorBg = colorize.CyanBg
		str = "[INFO]"
	case logrus.WarnLevel:
		colorBg = colorize.YellowBg
		str = "[WARN]"
	case logrus.ErrorLevel:
		colorBg = colorize.RedBg
		str = "[ERRO]"
	case logrus.FatalLevel:
		colorBg = colorize.RedBg
		str = "[FATL]"
	case logrus.PanicLevel:
		colorBg = colorize.RedBg
		str = "[PANC]"
	default:
		colorBg = colorize.BlackBg
	}

	str = colorize.ColorString(str, colorize.WhiteBold, colorBg)

	return
}

func initSender() {
	for _, sndr := range senders {
		sndr.Init()
	}

	go func() {
		for {
			entry := <-buffer

			if len(entry.Message) > 7 && entry.Message[:7] == "logger:" {
				continue
			}

			for _, sndr := range senders {
				sndr.Parse(entry)
			}
		}
	}()
}

func init() {
	module := requires.New("logger")

	module.Handler = func() {
		initSender()

		logrus.SetFormatter(&formatter{})
		logrus.AddHook(&logHook{})
		logrus.SetOutput(os.Stderr)
		logrus.SetLevel(logrus.InfoLevel)
	}
}
