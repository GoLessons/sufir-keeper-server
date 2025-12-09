package aead

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	aad := []byte("user|item|TEXT")
	pt := []byte("secret payload")
	nonce, ct, err := Encrypt(key, aad, pt)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Decrypt(key, aad, nonce, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, pt) {
		t.Fatalf("plaintext mismatch: got %q, want %q", out, pt)
	}
}

func TestDecryptWithWrongAAD(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	nonce, ct, err := Encrypt(key, []byte("a1"), []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(key, []byte("a2"), nonce, ct); err == nil {
		t.Fatal("expected authentication error with wrong AAD")
	}
}
