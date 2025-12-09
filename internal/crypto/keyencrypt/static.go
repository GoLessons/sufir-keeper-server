package keyencrypt

import (
	"context"
)

type StaticProvider struct {
	Key     []byte
	Version int
}

func (p *StaticProvider) GetCurrent(_ context.Context) ([]byte, int, error) {
	return p.Key, p.Version, nil
}
func (p *StaticProvider) GetByVersion(_ context.Context, _ int) ([]byte, error) { return p.Key, nil }
func (p *StaticProvider) Rotate(_ context.Context) ([]byte, int, error)         { return p.Key, p.Version, nil }
