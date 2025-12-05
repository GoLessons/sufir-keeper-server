package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	plain := "StrongPassword123!"
	hashed, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if len(hashed) == 0 {
		t.Fatalf("empty hash")
	}
	if !VerifyPassword(hashed, plain) {
		t.Fatalf("verify failed for correct password")
	}
	if VerifyPassword(hashed, "wrong") {
		t.Fatalf("verify should fail for wrong password")
	}
}
