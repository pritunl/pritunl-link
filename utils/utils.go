// Miscellaneous utilities.
package utils

import (
	"net"
	"os"
	"strings"

	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
)

func GetLocalAddress() (addr string, err error) {
	name, err := os.Hostname()
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "utils: Get ip"),
		}
		return
	}

	addrs, err := net.LookupHost(name)
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "utils: Get ip"),
		}
		return
	}

	addr = addrs[0]

	return
}

func StringSet(items []string) (s set.Set) {
	s = set.NewSet()

	for _, item := range items {
		s.Add(item)
	}

	return
}

func IncIpAddress(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func FormatHost(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}
