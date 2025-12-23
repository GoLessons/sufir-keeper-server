package s3

import (
	"context"
	"testing"
	"time"
)

func TestNewClientInvalidConfig(t *testing.T) {
	_, err := NewClient("", "ak", "sk", "bucket")
	if err == nil {
		t.Fatalf("expected error for empty endpoint")
	}
	_, err = NewClient("http://localhost:9000", "", "sk", "bucket")
	if err == nil {
		t.Fatalf("expected error for empty access key")
	}
	_, err = NewClient("http://localhost:9000", "ak", "", "bucket")
	if err == nil {
		t.Fatalf("expected error for empty secret key")
	}
	_, err = NewClient("http://localhost:9000", "ak", "sk", "")
	if err == nil {
		t.Fatalf("expected error for empty bucket")
	}
}

func TestNewClientBadURL(t *testing.T) {
	_, err := NewClient("::::", "ak", "sk", "bucket")
	if err == nil {
		t.Fatalf("expected error for malformed endpoint URL")
	}
}

func TestNewClientHTTPS(t *testing.T) {
	_, err := NewClient("https://localhost:9000", "ak", "sk", "bucket")
	if err != nil {
		t.Fatalf("unexpected error creating https client: %v", err)
	}
}

func TestPresignPostValidatesKeyAndMetadata(t *testing.T) {
	client, err := NewClient("http://localhost:9000", "ak", "sk", "bucket")
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}
	_, _, err = client.PresignPost(context.Background(), "", "application/octet-stream", 0, map[string]string{"x": "y"}, time.Minute)
	if err == nil {
		t.Fatalf("expected error for empty key")
	}
}
