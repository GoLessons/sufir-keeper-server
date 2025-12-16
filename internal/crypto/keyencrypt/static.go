package keyencrypt

import (
	"context"
)

// Стаб для подмены в тестах
type StaticProvider struct {
	Key     []byte
	Version int
}

func NewStaticProvider(key []byte, version int) *StaticProvider {
	return &StaticProvider{Key: key, Version: version}
}

func (p *StaticProvider) GetCurrent(_ context.Context) ([]byte, int, error) {
	return p.Key, p.Version, nil
}
func (p *StaticProvider) GetByVersion(_ context.Context, _ int) ([]byte, error) { return p.Key, nil }
func (p *StaticProvider) Rotate(_ context.Context) ([]byte, int, error)         { return p.Key, p.Version, nil }
