package status

import (
	"strconv"
	"strings"
	"time"

	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
)

type Status map[string]string

func (s Status) Merge(other Status) Status {
	if s == nil {
		return other
	}
	if other == nil {
		return s
	}

	result := Status{}
	for key, value := range s {
		result[key] = value
	}

	for key, value := range other {
		result[key] = value
	}

	return result
}

func Get() (status Status, err error) {
	status = Status{}

	output, err := utils.ExecOutput("", "ipsec", "status")
	if err != nil {
		err = nil
		return
	}

	isIkeState := false
	ikeState := ""

	for _, line := range strings.Split(output, "\n") {
		lines := strings.SplitN(line, ":", 2)
		if len(lines) != 2 {
			continue
		}

		isIkeState = strings.HasSuffix(lines[0], "]")

		if isIkeState {
			ikeState = strings.SplitN(
				strings.TrimSpace(lines[1]), " ", 2)[0]
		} else {
			if !strings.Contains(lines[1], "reqid") {
				continue
			}

			if !strings.Contains(lines[0], "{") {
				continue
			}

			connId := strings.SplitN(lines[0], "{", 2)[0]
			connState := strings.SplitN(
				strings.TrimSpace(lines[1]), ",", 2)[0]

			switch ikeState {
			case "ESTABLISHED":
				if connState == "INSTALLED" {
					connState = "connected"
				} else {
					connState = "disconnected"
				}
				break
			case "CONNECTING":
				connState = "connecting"
			default:
				connState = "disconnected"
			}

			curState := status[connId]
			if curState == "" || curState == "disconnected" ||
				(curState == "connecting" && connState == "connected") {

				status[connId] = connState
			}
		}
	}

	return
}

func GetWg(wgKeyMap map[string]string) (status Status, err error) {
	status = Status{}

	output, err := utils.ExecOutput("", "wg", "show", "all", "dump")
	if err != nil {
		err = nil
		return
	}

	now := time.Now()

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 8 {
			continue
		}

		pubKey := fields[1]
		connId := wgKeyMap[pubKey]
		if connId == "" {
			continue
		}

		handshakeUnix, _ := strconv.ParseInt(fields[5], 10, 64)
		var handshakeTime time.Time
		if handshakeUnix > 0 {
			handshakeTime = time.Unix(handshakeUnix, 0)
		}

		connState := ""
		if now.Sub(handshakeTime) < 3*time.Minute {
			connState = "connected"
		} else {
			connState = "disconnected"
		}

		status[connId] = connState
	}

	return
}

func GetIds() (connIds []string, err error) {
	connIds = []string{}
	connIdsSet := set.NewSet()

	output, err := utils.ExecOutput("", "ipsec", "status")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"output": output,
			"error":  err,
		}).Warn("status: Failed to get ipsec status ids")
		err = nil
		return
	}

	for _, line := range strings.Split(output, "\n") {
		lines := strings.SplitN(line, ":", 2)
		if len(lines) != 2 {
			continue
		}

		connId := strings.Split(lines[0], "[")[0]
		connId = strings.Split(connId, "{")[0]
		connIdSpl := strings.Split(connId, "-")
		if len(connIdSpl) == 3 && len(connIdSpl[0]) == 24 {
			if !connIdsSet.Contains(connId) {
				connIdsSet.Add(connId)
				connIds = append(connIds, connId)
			}
		}
	}

	return
}
