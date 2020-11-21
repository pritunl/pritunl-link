package oracle

import (
	"bytes"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
)

func oracleParseBase64Key(data string) (key *rsa.PrivateKey,
	fingerprint string, err error) {

	pemKey, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "oracle: Failed to parse base64 private key"),
		}
		return
	}
	block, _ := pem.Decode(pemKey)
	if block == nil {
		err = &errortypes.ParseError{
			errors.New("oracle: Failed to decode private key"),
		}
		return
	}

	if block.Type != "RSA PRIVATE KEY" {
		err = &errortypes.ParseError{
			errors.New("oracle: Invalid private key type"),
		}
		return
	}

	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to parse rsa key"),
		}
		return
	}

	pubKey, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to marshal public key"),
		}
		return
	}

	keyHash := md5.New()
	keyHash.Write(pubKey)
	fingerprint = fmt.Sprintf("%x", keyHash.Sum(nil))
	fingerprintBuf := bytes.Buffer{}

	for i, run := range fingerprint {
		fingerprintBuf.WriteRune(run)
		if i%2 == 1 && i != len(fingerprint)-1 {
			fingerprintBuf.WriteRune(':')
		}
	}
	fingerprint = fingerprintBuf.String()

	return
}
