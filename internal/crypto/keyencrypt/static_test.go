package keyencrypt

import (
	"context"
	"testing"
)

func TestStaticProviderAllMethods(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	p := NewStaticProvider(key, 3)
	curKey, curVer, err := p.GetCurrent(context.Background())
	if err != nil {
		t.Fatalf("GetCurrent error: %v", err)
	}
	if curVer != 3 {
		t.Fatalf("unexpected version: %d", curVer)
	}
	if len(curKey) != 32 {
		t.Fatalf("unexpected key size: %d", len(curKey))
	}
	key2, err := p.GetByVersion(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetByVersion error: %v", err)
	}
	if len(key2) != 32 {
		t.Fatalf("unexpected key size get by version: %d", len(key2))
	}
	key3, ver3, err := p.Rotate(context.Background())
	if err != nil {
		t.Fatalf("Rotate error: %v", err)
	}
	if ver3 != 3 {
		t.Fatalf("unexpected rotate version: %d", ver3)
	}
	if len(key3) != 32 {
		t.Fatalf("unexpected rotate key size: %d", len(key3))
	}
}
