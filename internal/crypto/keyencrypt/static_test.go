package keyencrypt

import (
	"context"
	"testing"
)

func TestStaticProvider(t *testing.T) {
	key := make([]byte, 32)
	p := NewStaticProvider(key, 1)
	if p == nil {
		t.Fatalf("provider nil")
	}
	k, v, err := p.GetCurrent(context.Background())
	if err != nil || v != 1 || len(k) != 32 {
		t.Fatalf("GetCurrent unexpected: %v %d %d", err, v, len(k))
	}
	k2, err := p.GetByVersion(context.Background(), 1)
	if err != nil || len(k2) != 32 {
		t.Fatalf("GetByVersion unexpected: %v %d", err, len(k2))
	}
	k3, v3, err := p.Rotate(context.Background())
	if err != nil || v3 != 1 || len(k3) != 32 {
		t.Fatalf("Rotate unexpected: %v %d %d", err, v3, len(k3))
	}
}
