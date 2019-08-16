package eciesgo

import (
	"crypto/sha256"
	"github.com/pkg/errors"
	"golang.org/x/crypto/hkdf"
	"io"
)

func kdf(secret []byte) (key []byte, err error) {
	key = make([]byte, 32)
	kdf := hkdf.New(sha256.New, secret, nil, nil)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, errors.Wrap(err, "cannot read secret from HKDF reader")
	}

	return key, nil
}