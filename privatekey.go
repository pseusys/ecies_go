package eciesgo

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
)

// PrivateKey is an instance of secp256k1 private key with nested public key
type PrivateKey struct {
	*PublicKey
	D *big.Int
}

// GenerateKey generates secp256k1 key pair
func GenerateKey() (*PrivateKey, error) {
	curve := getCurve()

	p, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cannot generate key pair: %w", err)
	}

	return &PrivateKey{
		PublicKey: &PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(p),
	}, nil
}

// NewPrivateKeyFromHex decodes hex form of private key raw bytes, computes public key and returns PrivateKey instance
func NewPrivateKeyFromHex(s string) (*PrivateKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("cannot decode hex string: %w", err)
	}

	return NewPrivateKeyFromBytes(b), nil
}

// NewPrivateKeyFromBytes decodes private key raw bytes, computes public key and returns PrivateKey instance
func NewPrivateKeyFromBytes(priv []byte) *PrivateKey {
	curve := getCurve()
	x, y := curve.ScalarBaseMult(priv)

	return &PrivateKey{
		PublicKey: &PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(priv),
	}
}

// Bytes returns private key raw bytes
func (k *PrivateKey) Bytes() []byte {
	return k.D.Bytes()
}

// Hex returns private key bytes in hex form
func (k *PrivateKey) Hex() string {
	return hex.EncodeToString(k.Bytes())
}

// Encapsulate encapsulates key by using Key Encapsulation Mechanism and returns symmetric key;
// can be safely used as encryption key
func (k *PrivateKey) Encapsulate(pub *PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, fmt.Errorf("public key is empty")
	}

	if !k.Curve.IsOnCurve(pub.X, pub.Y) {
		return nil, fmt.Errorf("invalid public key")
	}

	var secret bytes.Buffer
	secret.Write(k.PublicKey.Bytes(false))

	sx, sy := pub.Curve.ScalarMult(pub.X, pub.Y, k.D.Bytes())
	secret.Write([]byte{0x04})

	// Sometimes shared secret coordinates are less than 32 bytes; Big Endian
	l := len(pub.Curve.Params().P.Bytes())
	secret.Write(zeroPad(sx.Bytes(), l))
	secret.Write(zeroPad(sy.Bytes(), l))

	return kdf(secret.Bytes())
}

// ECDH derives shared secret;
// Must not be used as encryption key, it increases chances to perform successful key restoration attack
func (k *PrivateKey) ECDH(pub *PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, fmt.Errorf("public key is empty")
	}

	if !k.Curve.IsOnCurve(pub.X, pub.Y) {
		return nil, fmt.Errorf("invalid public key")
	}

	// Shared secret generation
	sx, sy := pub.Curve.ScalarMult(pub.X, pub.Y, k.D.Bytes())

	var ss []byte
	if sy.Bit(0) != 0 { // If odd
		ss = append(ss, 0x03)
	} else { // If even
		ss = append(ss, 0x02)
	}

	// Sometimes shared secret is less than 32 bytes; Big Endian
	l := len(pub.Curve.Params().P.Bytes())
	for i := 0; i < l-len(sx.Bytes()); i++ {
		ss = append(ss, 0x00)
	}

	return append(ss, sx.Bytes()...), nil
}

// Equals compares two private keys with constant time (to resist timing attacks)
func (k *PrivateKey) Equals(priv *PrivateKey) bool {
	return subtle.ConstantTimeCompare(k.D.Bytes(), priv.D.Bytes()) == 1
}
