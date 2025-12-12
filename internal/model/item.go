package model

import (
	"time"

	"github.com/google/uuid"
)

type ItemFile struct {
	S3Bucket string
	S3Key    string
	SHA256   string
	Size     int64
}

type ItemRecord struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Meta             map[string]string
	File             *ItemFile
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
