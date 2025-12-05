package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	CreatedAt    time.Time
	Login        string
	PasswordHash string
	ID           uuid.UUID
}

func NewUser(id uuid.UUID, login string, passwordHash string, createdAt time.Time) User {
	return User{ID: id, Login: login, PasswordHash: passwordHash, CreatedAt: createdAt}
}
