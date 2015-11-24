// Miscellaneous utilities.
package utils

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-auth/constants"
	"math/rand"
	"net"
	"os"
)

var (
	chars = []rune(
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func RandStr(n int) (str string) {
	strList := make([]rune, n)
	for i := range strList {
		strList[i] = chars[rand.Intn(len(chars))]
	}
	str = string(strList)
	return
}

func GetLocalAddress() (addr string, err error) {
	name, err := os.Hostname()
	if err != nil {
		err = &constants.UnknownError{
			errors.Wrap(err, "utils: Get ip"),
		}
		return
	}

	addrs, err := net.LookupHost(name)
	if err != nil {
		err = &constants.UnknownError{
			errors.Wrap(err, "utils: Get ip"),
		}
		return
	}

	addr = addrs[0]

	return
}
