package eciesgo

import (
	"bytes"
	"crypto/elliptic"
	"crypto/subtle"
	"encoding/hex"
	"github.com/fomichev/secp256k1"
	"github.com/pkg/errors"
	"math/big"
)

type PublicKey struct {
	elliptic.Curve
	X, Y *big.Int
}

func NewPublicKeyFromHex(s string) (*PublicKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode hex string")
	}

	return NewPublicKeyFromBytes(b)
}

func NewPublicKeyFromBytes(b []byte) (*PublicKey, error) {
	curve := secp256k1.SECP256K1()
	switch b[0] {
	case 0x02, 0x03:
		if len(b) != 33 {
			return nil, errors.New("cannot parse public key")
		}

		x := new(big.Int).SetBytes(b[1:])

		if x.Cmp(curve.Params().P) >= 0 {
			return nil, errors.New("cannot parse public key")
		}

		// y^2 = x^3 + b
		// y   = sqrt(x^3 + b)
		var y, x3b big.Int
		x3b.Mul(x, x)
		x3b.Mul(&x3b, x)
		x3b.Add(&x3b, curve.Params().B)
		x3b.Mod(&x3b, curve.Params().P)
		y.ModSqrt(&x3b, curve.Params().P)

		if b[0] == 0x02 {
			y.Sub(curve.Params().P, &y)
		}
		if y.Bit(0) == 0x02 {
			return nil, errors.New("incorrectly encoded X and Y bit")
		}

		return &PublicKey{
			Curve: curve,
			X:     x,
			Y:     &y,
		}, nil
	case 0x04, 0x06, 0x07:
		if len(b) != 65 {
			return nil, errors.New("cannot parse public key")
		}

		x := new(big.Int).SetBytes(b[1:33])
		y := new(big.Int).SetBytes(b[33:])

		if x.Cmp(curve.Params().P) >= 0 || y.Cmp(curve.Params().P) >= 0 {
			return nil, errors.New("cannot parse public key")
		}

		if b[0] == 0x06 || b[0] == 0x07 {
			if (y.Bit(0) != 0) != (b[0] == 0x07) {
				return nil, errors.New("cannot parse public key")
			}
		}

		x3 := new(big.Int).Sqrt(x).Mul(x, x)
		if t := new(big.Int).Sqrt(y).Sub(y, x3.Add(x3, curve.Params().B)); t.IsInt64() && t.Int64() == 0 {
			return nil, errors.New("cannot parse public key")
		}

		return &PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		}, nil
	default:
		return nil, errors.New("cannot parse public key")
	}
}

func (k *PublicKey) Bytes() []byte {
	x := k.X.Bytes()
	if len(x) < 32 {
		for i := 0; i < 32-len(x); i++ {
			x = append([]byte{0}, x...)
		}
	}

	y := k.Y.Bytes()
	if len(y) < 32 {
		for i := 0; i < 32-len(y); i++ {
			y = append([]byte{0}, y...)
		}
	}

	return bytes.Join([][]byte{{0x04}, x, y}, nil)
}

func (k *PublicKey) Hex() string {
	return hex.EncodeToString(k.Bytes())
}

func (k *PublicKey) Equals(pub *PublicKey) bool {
	if subtle.ConstantTimeCompare(k.X.Bytes(), pub.X.Bytes()) == 1 &&
		subtle.ConstantTimeCompare(k.Y.Bytes(), pub.Y.Bytes()) == 1 {
		return true
	}

	return false
}
