package logger

import (
	"github.com/Sirupsen/logrus"
)

type apiSender struct{}

func (a *apiSender) Init() {}

func (a *apiSender) Parse(entry *logrus.Entry) {
	msg := formatPlain(entry)
	// TODO
	_ = msg
}

func init() {
	senders = append(senders, &apiSender{})
}
