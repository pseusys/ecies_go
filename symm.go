package eciesgo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

func generateSymmCipher(key []byte, conf Config) (cipher.AEAD, error) {
	var err error
	var aead cipher.AEAD

	switch conf.symmetricAlgorithm {
	case "aes-256-gcm":
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("cannot create new AES block: %w", err)
		}

		aead, err = cipher.NewGCMWithNonceSize(block, conf.symmetricNonceLength)
		if err != nil {
			return nil, fmt.Errorf("cannot create AES GCM: %w", err)
		}
	case "xchacha20":
		aead, err = chacha20poly1305.NewX(key)
		if err != nil {
			return nil, fmt.Errorf("cannot create XChaCha20: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown cipher: %s", conf.symmetricAlgorithm)
	}

	return aead, nil
}

func EncryptSymm(key []byte, msg []byte, conf Config) ([]byte, error) {
	var ct bytes.Buffer

	aead, err := generateSymmCipher(key, conf)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("cannot read random bytes for nonce: %w", err)
	}

	ct.Write(nonce)

	ciphertext := aead.Seal(nil, nonce, msg, nil)

	tag := ciphertext[len(ciphertext)-aead.Overhead():]
	ct.Write(tag)
	ciphertext = ciphertext[:len(ciphertext)-len(tag)]
	ct.Write(ciphertext)

	return ct.Bytes(), nil
}

func DecryptSymm(key []byte, msg []byte, conf Config) ([]byte, error) {
	aead, err := generateSymmCipher(key, conf)
	if err != nil {
		return nil, err
	}

	// Message cannot be less than length of public key (65) + nonce + tag (16)
	if len(msg) <= (aead.NonceSize() + aead.Overhead()) {
		return nil, fmt.Errorf("invalid length of message")
	}

	// Symmetrical decryption part
	nonce := msg[:aead.NonceSize()]
	tag := msg[aead.NonceSize() : aead.NonceSize()+aead.Overhead()]
	msg = msg[aead.NonceSize()+aead.Overhead():]

	// Create Golang-accepted ciphertext
	ciphertext := bytes.Join([][]byte{msg, tag}, nil)

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt ciphertext: %v", err)
	}

	return plaintext, nil
}
