package utils

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"math/big"
	mathrand "math/rand"
)

func RandBytes(size int) (bytes []byte, err error) {
	bytes = make([]byte, size)
	_, err = rand.Read(bytes)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "utils: Random read error"),
		}
		return
	}

	return
}

func Uuid() (id string) {
	data, err := RandBytes(16)
	if err != nil {
		panic(err)
	}

	id = hex.EncodeToString(data[:])
	return
}

func init() {
	n, err := rand.Int(rand.Reader, big.NewInt(9223372036854775806))
	if err != nil {
		panic(err)
	}

	mathrand.Seed(n.Int64())
}
