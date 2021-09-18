package interlink

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
)

var (
	client = &http.Client{
		Timeout: 3 * time.Second,
	}
)

func CheckHost(addr string) (state bool, latency int, err error) {
	if strings.Contains(addr, ":") {
		addr = "[" + addr + "]"
	}

	u := url.URL{
		Scheme: "http",
		Host:   addr + ":9790",
		Path:   "/check",
	}

	req, err := http.NewRequest(
		"GET",
		u.String(),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "interlink: Request init error"),
		}
		return
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "interlink: Request put error"),
		}
		return
	}
	defer resp.Body.Close()
	latency = int(time.Since(start).Microseconds())

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Newf("interlink: Request bad status %d", resp.StatusCode),
		}
		return
	}

	state = true

	return
}
