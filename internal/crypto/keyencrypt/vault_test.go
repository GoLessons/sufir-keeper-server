package keyencrypt

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/vault/api"
)

const vaultKVDataPath = "/v1/secret/data/keys"

func TestVaultProviderInitializeAndGetCurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == vaultKVDataPath {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) && r.URL.Path == vaultKVDataPath {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"data": {}}}`))
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := api.DefaultConfig()
	cfg.Address = server.URL
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	client.SetToken("token")
	masterKeyHex := "0000000000000000000000000000000000000000000000000000000000000000"

	provider, err := NewVaultProviderWithClient(client, "keys", masterKeyHex)
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	key, version, err := provider.GetCurrent(t.Context())
	if err != nil {
		t.Fatalf("get current error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("unexpected key length: %d", len(key))
	}
	if version != 1 {
		t.Fatalf("unexpected version: %d", version)
	}
}

func TestVaultProviderGetByVersionMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == vaultKVDataPath {
			body := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"current": 1,
						"versions": map[string]interface{}{
							"1": map[string]interface{}{"ciphertext": "00", "nonce": "00"},
						},
					},
				},
			}
			buf, _ := json.Marshal(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf)
			return
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) && r.URL.Path == vaultKVDataPath {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"data": {}}}`))
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := api.DefaultConfig()
	cfg.Address = server.URL
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	client.SetToken("token")
	masterKeyHex := "0000000000000000000000000000000000000000000000000000000000000000"
	provider, err := NewVaultProviderWithClient(client, "keys", masterKeyHex)
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	_, err = provider.GetByVersion(t.Context(), 3)
	if err == nil {
		t.Fatalf("expected error when version entry missing")
	}
}

func TestNewVaultProviderWithClientBadMasterKey(t *testing.T) {
	cfg := api.DefaultConfig()
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	_, err = NewVaultProviderWithClient(client, "keys", "deadbeef")
	if err == nil {
		t.Fatalf("expected error for bad master key hex")
	}
}

func TestVaultProviderGetByVersionInvalidHex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == vaultKVDataPath {
			body := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"current": 1,
						"versions": map[string]interface{}{
							"1": map[string]interface{}{"ciphertext": "zz", "nonce": "00"},
						},
					},
				},
			}
			buf, _ := json.Marshal(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf)
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := api.DefaultConfig()
	cfg.Address = server.URL
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	client.SetToken("token")
	masterKeyHex := "0000000000000000000000000000000000000000000000000000000000000000"
	provider, err := NewVaultProviderWithClient(client, "keys", masterKeyHex)
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	_, err = provider.GetByVersion(t.Context(), 1)
	if err == nil {
		t.Fatalf("expected error for invalid ciphertext hex")
	}
}

func TestVaultProviderRotateCurrentAsStringInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == vaultKVDataPath {
			body := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"current":  "not-an-int",
						"versions": map[string]interface{}{},
					},
				},
			}
			buf, _ := json.Marshal(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf)
			return
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) && r.URL.Path == vaultKVDataPath {
			var incoming map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&incoming)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"data": {}}}`))
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := api.DefaultConfig()
	cfg.Address = server.URL
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	client.SetToken("token")
	masterKeyHex := "0000000000000000000000000000000000000000000000000000000000000000"
	provider, err := NewVaultProviderWithClient(client, "keys", masterKeyHex)
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	_, ver, err := provider.Rotate(t.Context())
	if err != nil {
		t.Fatalf("rotate error: %v", err)
	}
	if ver != 1 {
		t.Fatalf("unexpected version after rotate: %d", ver)
	}
}

func TestVaultProviderRotateAndGetByVersion(t *testing.T) {
	created := false
	storage := map[string]interface{}{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == vaultKVDataPath {
			if !created {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			body := map[string]interface{}{
				"data": map[string]interface{}{
					"data": storage,
				},
			}
			buf, _ := json.Marshal(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf)
			return
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) && r.URL.Path == vaultKVDataPath {
			var incoming map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&incoming)
			created = true
			body := map[string]interface{}{
				"data": map[string]interface{}{
					"data": incoming,
				},
			}
			if data := incoming; data != nil {
				if d, ok := data["data"].(map[string]interface{}); ok {
					storage = d
				}
			}
			buf, _ := json.Marshal(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf)
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := api.DefaultConfig()
	cfg.Address = server.URL
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	client.SetToken("token")
	masterKeyHex := "0000000000000000000000000000000000000000000000000000000000000000"
	provider, err := NewVaultProviderWithClient(client, "keys", masterKeyHex)
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}

	key1, version1, err := provider.GetCurrent(t.Context())
	if err != nil {
		t.Fatalf("get current error: %v", err)
	}
	key2, version2, err := provider.Rotate(t.Context())
	if err != nil {
		t.Fatalf("rotate error: %v", err)
	}
	if version1 != 1 || version2 != 2 {
		t.Fatalf("unexpected versions: %d, %d", version1, version2)
	}
	got1, err := provider.GetByVersion(t.Context(), 1)
	if err != nil {
		t.Fatalf("get by version error: %v", err)
	}
	if len(got1) != len(key1) {
		t.Fatalf("unexpected key length on get by version")
	}
	if bytes.Equal(got1, key2) && bytes.Equal(key1, key2) {
		t.Fatalf("expected rotated key to differ")
	}
}
