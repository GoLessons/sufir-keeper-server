package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestClientClose_NilSafe(t *testing.T) {
	var c *Client
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error for nil client, got %v", err)
	}
	c = &Client{}
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error for nil SQL, got %v", err)
	}
}

func TestClientClose_EmptyDB(_ *testing.T) {
	// Create a sql.DB via an in-memory DSN not available; focus on method behavior.
	// We cannot call NewClient without Postgres available; use a dummy sql.DB.
	c := &Client{SQL: &sql.DB{}}
	if err := c.Close(); err == nil {
		// Close on zero sql.DB returns error; both nil and non-nil are acceptable here.
		// The key is the method does not panic.
		_ = c
	}
}

func TestNewClient_InvalidDSN(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, "invalid-dsn", Options{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: time.Second, ConnMaxIdleTime: time.Second})
	if err == nil {
		t.Fatalf("expected error for invalid dsn")
	}
}
