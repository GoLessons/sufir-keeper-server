package keyencrypt

import (
	"context"
)

type Provider interface {
	GetCurrent(ctx context.Context) ([]byte, int, error)
	GetByVersion(ctx context.Context, version int) ([]byte, error)
	Rotate(ctx context.Context) ([]byte, int, error)
}
