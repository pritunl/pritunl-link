package profile

import (
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"strings"
)

var (
	Host         string
	Token        string
	Secret       string
	Username     string
	NetworkLinks []string
)

type Profile struct {
	UserId         string   `json:"user_id"`
	OrganizationId string   `json:"organization_id"`
	ServerId       string   `json:"server_id"`
	SyncHash       string   `json:"sync_hash"`
	SyncToken      string   `json:"sync_token"`
	SyncSecret     string   `json:"sync_secret"`
	SyncHosts      []string `json:"sync_hosts"`
	Conf           string   `json:"conf"`
}

func (p *Profile) Parse(data string) (err error) {
	lines := strings.Split(data, "\n")
	jsonData := ""
	conf := ""

	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			jsonData += strings.TrimSpace(line[1:])
		} else {
			conf = strings.Join(lines[i:], "\n")
			break
		}
	}

	err = json.Unmarshal([]byte(jsonData), p)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "profile: Failed to parse json data"),
		}
		return
	}

	p.Conf = conf

	return
}
