package aead

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptRejectsInvalidKeyLength(t *testing.T) {
	shortKey := make([]byte, 16)
	aad := []byte("aad")
	plaintext := []byte("data")
	nonce, ciphertext, err := Encrypt(shortKey, aad, plaintext)
	if err == nil {
		t.Fatalf("expected error for short key, got nil (nonce=%v, ct=%v)", nonce, ciphertext)
	}
	longKey := make([]byte, 64)
	if _, err := rand.Read(longKey); err != nil {
		t.Fatalf("rand.Read error: %v", err)
	}
	nonce, ciphertext, err = Encrypt(longKey, aad, plaintext)
	if err == nil {
		t.Fatalf("expected error for long key, got nil (nonce=%v, ct=%v)", nonce, ciphertext)
	}
}

func TestDecryptRejectsInvalidKeyLength(t *testing.T) {
	validKey := make([]byte, 32)
	if _, err := rand.Read(validKey); err != nil {
		t.Fatalf("rand.Read error: %v", err)
	}
	aad := []byte("aad")
	plaintext := []byte("payload")
	nonce, ciphertext, err := Encrypt(validKey, aad, plaintext)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	shortKey := make([]byte, 16)
	if _, err := Decrypt(shortKey, aad, nonce, ciphertext); err == nil {
		t.Fatalf("expected error for short key in decrypt, got nil")
	}
	longKey := make([]byte, 64)
	if _, err := rand.Read(longKey); err != nil {
		t.Fatalf("rand.Read error: %v", err)
	}
	if _, err := Decrypt(longKey, aad, nonce, ciphertext); err == nil {
		t.Fatalf("expected error for long key in decrypt, got nil")
	}
}

func TestEncryptDecryptLargePayload(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read error: %v", err)
	}
	aad := []byte("large|payload|aad")
	size := 1024 * 64
	plaintext := make([]byte, size)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatalf("rand.Read error: %v", err)
	}
	nonce, ciphertext, err := Encrypt(key, aad, plaintext)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatalf("ciphertext is empty for large payload")
	}
	decrypted, err := Decrypt(key, aad, nonce, ciphertext)
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("mismatch for large payload")
	}
}
