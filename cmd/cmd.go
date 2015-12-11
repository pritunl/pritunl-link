// Commands available in cli.
package cmd

import (
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/utils"
	"os"
)

type options struct {
	Id      string
	Host    string
	Token   string
	Secret  string
	ConfDir string
}

func getOptions() (opts *options) {
	id := os.Getenv("ID")
	if id == "" {
		id = utils.RandName()
	}

	constants.Id = id

	opts = &options{
		Id:      id,
		Host:    os.Getenv("HOST"),
		Token:   os.Getenv("TOKEN"),
		Secret:  os.Getenv("SECRET"),
		ConfDir: os.Getenv("CONF_DIR"),
	}

	return
}
