package testutil

import (
	"context"
)

type StaticKeyEncryptProvider struct {
	Key     []byte
	Version int
}

func NewStaticKeyEncryptProvider(key []byte, version int) *StaticKeyEncryptProvider {
	return &StaticKeyEncryptProvider{Key: key, Version: version}
}

func (provider *StaticKeyEncryptProvider) GetCurrent(_ context.Context) ([]byte, int, error) {
	return provider.Key, provider.Version, nil
}

func (provider *StaticKeyEncryptProvider) GetByVersion(_ context.Context, _ int) ([]byte, error) {
	return provider.Key, nil
}

func (provider *StaticKeyEncryptProvider) Rotate(_ context.Context) ([]byte, int, error) {
	return provider.Key, provider.Version, nil
}
