package keyencrypt

import (
	"testing"

	"github.com/hashicorp/vault/api"
)

func TestNewVaultProviderInvalidMasterKey(t *testing.T) {
	if _, err := NewVaultProvider("", "", "path", "zz"); err == nil {
		t.Fatalf("expected error for invalid hex")
	}
	if _, err := NewVaultProvider("", "", "path", "01"); err == nil {
		t.Fatalf("expected error for short hex")
	}
	if _, err := NewVaultProvider("", "", "path", "0102030405060708090a0b0c0d0e0f10"); err == nil {
		t.Fatalf("expected error for 16-byte hex")
	}
}

func TestNewVaultProviderWithClientInvalidMasterKey(t *testing.T) {
	c, _ := api.NewClient(api.DefaultConfig())
	if _, err := NewVaultProviderWithClient(c, "kv", "zz"); err == nil {
		t.Fatalf("expected error for invalid hex")
	}
	if _, err := NewVaultProviderWithClient(c, "kv", "01"); err == nil {
		t.Fatalf("expected error for short hex")
	}
	if _, err := NewVaultProviderWithClient(c, "kv", "0102030405060708090a0b0c0d0e0f10"); err == nil {
		t.Fatalf("expected error for 16-byte hex")
	}
}
