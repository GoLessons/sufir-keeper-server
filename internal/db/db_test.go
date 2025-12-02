package db

import (
	"context"
	"testing"
)

func TestNewClientInvalidDsn(t *testing.T) {
	_, err := NewClient(context.Background(), "postgres://invalid-host-name-that-should-fail:5432/db", Options{})
	if err == nil {
		t.Fatalf("expected error for invalid connection, got nil")
	}
}
