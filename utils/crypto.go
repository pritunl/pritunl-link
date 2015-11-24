package utils

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/pritunl/pritunl-auth/errortypes"
	"github.com/dropbox/godropbox/errors"
	"math/big"
	mathrand "math/rand"
	"time"
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

func seedRand() {
	n, err := rand.Int(rand.Reader, big.NewInt(9223372036854775806))
	if err != nil {
		mathrand.Seed(time.Now().UTC().UnixNano())
		return
	}

	mathrand.Seed(n.Int64())
	return
}

func init() {
	seedRand()
}
