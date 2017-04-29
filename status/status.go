package status

import (
	"github.com/pritunl/pritunl-link/utils"
	"strings"
)

var Status = map[string]string{}

func Update() (status map[string]string, err error) {
	status = map[string]string{}

	output, err := utils.ExecOutput("", "ipsec", "status")
	if err != nil {
		return
	}

	for _, line := range strings.Split(output, "\n") {
		lines := strings.SplitN(line, ":", 2)
		if len(lines) != 2 {
			continue
		}

		if !strings.HasSuffix(lines[0], "]") {
			continue
		}

		connId := strings.SplitN(lines[0], "[", 2)[0]
		connState := strings.SplitN(
			strings.TrimSpace(lines[1]), " ", 2)[0]

		switch connState {
		case "ESTABLISHED":
			connState = "connected"
			break
		case "CONNECTING":
			connState = "connecting"
			break
		default:
			connState = "disconnected"
		}

		if _, ok := status[connId]; !ok {
			status[connId] = connState
		}
	}

	Status = status

	return
}
