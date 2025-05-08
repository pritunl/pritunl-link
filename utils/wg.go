package utils

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"golang.org/x/crypto/curve25519"
)

const KeyLen = 32

type Key [KeyLen]byte

func (k Key) PublicKey() (key Key) {
	var pub [KeyLen]byte
	var priv = [KeyLen]byte(k)

	curve25519.ScalarBaseMult(&pub, &priv)

	key = Key(pub)
	return
}

func (k Key) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

func GeneratePrivateKey() (Key, error) {
	key, err := GenerateKey()
	if err != nil {
		return Key{}, err
	}

	key[0] &= 248
	key[31] &= 127
	key[31] |= 64

	return key, nil
}

func GenerateKey() (key Key, err error) {
	b := make([]byte, 32)

	_, err = rand.Read(b)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "link: Failed to read random bytes"),
		}
		return
	}

	return NewKey(b)
}

func NewKey(b []byte) (key Key, err error) {
	if len(b) != KeyLen {
		err = &errortypes.ParseError{
			errors.Newf("link: incorrect key size: %d", len(b)),
		}
		return
	}

	copy(key[:], b)
	return
}
