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

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		password string
		valid    bool
	}{
		{password: "short!", valid: false},
		{password: "LongButNoSpecials123", valid: false},
		{password: "Contains Special!123", valid: true},
		{password: "With Space And !", valid: true},
		{password: "aaaaaaaa", valid: false},
	}
	for _, c := range cases {
		ok := ValidatePasswordStrength(c.password)
		if ok != c.valid {
			t.Fatalf("unexpected result for %q: got %v, want %v", c.password, ok, c.valid)
		}
	}
}
