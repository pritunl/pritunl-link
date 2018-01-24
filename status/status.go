package status

import (
	"github.com/pritunl/pritunl-link/utils"
	"strings"
)

type Status map[string]map[string]string

func Get() (status Status, connected int, err error) {
	connected = 0
	status = Status{}

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

		connId := strings.SplitN(strings.SplitN(lines[0], "[", 2)[0], "-", 2)
		connState := strings.SplitN(
			strings.TrimSpace(lines[1]), " ", 2)[0]

		if len(connId) != 2 {
			continue
		}

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

		if _, ok := status[connId[0]]; !ok {
			status[connId[0]] = map[string]string{}
		}

		if _, ok := status[connId[0]][connId[1]]; !ok {
			status[connId[0]][connId[1]] = connState
		} else if (status[connId[0]][connId[1]] == "disconnected") ||
			(status[connId[0]][connId[1]] == "connecting" &&
				connState == "connected") {

			status[connId[0]][connId[1]] = connState
		}
	}

	for _, stat := range status {
		for _, conn := range stat {
			if conn == "connected" {
				connected += 1
			}
		}
	}

	return
}
