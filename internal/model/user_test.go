package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUserInitializesFields(t *testing.T) {
	userID := uuid.New()
	login := "login"
	passwordHash := "hash"
	createdAt := time.Now().UTC()

	user := NewUser(userID, login, passwordHash, createdAt)

	if user.ID != userID {
		t.Fatalf("unexpected id")
	}
	if user.Login != login {
		t.Fatalf("unexpected login")
	}
	if user.PasswordHash != passwordHash {
		t.Fatalf("unexpected password hash")
	}
	if !user.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected created at")
	}
}
