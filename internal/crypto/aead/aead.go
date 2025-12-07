package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

func Encrypt(key []byte, aad []byte, plaintext []byte) ([]byte, []byte, error) {
	if len(key) != 32 {
		return nil, nil, errors.New("invalid key length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, aad)
	return nonce, sealed, nil
}

func Decrypt(key []byte, aad []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}
