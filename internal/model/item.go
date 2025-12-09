package model

import (
	"time"

	"github.com/google/uuid"
)

type ItemRecord struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Meta             map[string]string
	Title            string
	Type             string
	DataEncrypted    []byte
	DataNonce        []byte
	DataKeyEncrypted []byte
	DataKeyNonce     []byte
	KEKVersion       int
	ID               uuid.UUID
	UserID           uuid.UUID
}

type ItemListRecord struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Meta      map[string]string
	Title     string
	ID        uuid.UUID
}
