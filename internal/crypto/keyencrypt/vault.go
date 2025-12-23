package keyencrypt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/vault/api"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
)

type VaultProvider struct {
	client    *api.Client
	kvPath    string
	masterKey []byte
}

func NewVaultProvider(addr string, token string, kvPath string, masterKeyHex string) (*VaultProvider, error) {
	cfg := api.DefaultConfig()
	if err := cfg.ConfigureTLS(&api.TLSConfig{Insecure: true}); err != nil {
		return nil, err
	}
	if addr != "" {
		cfg.Address = addr
	}
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	c.SetToken(token)
	mk, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, err
	}
	if len(mk) != 32 {
		return nil, errors.New("master key must be 32 bytes (HEX)")
	}
	return &VaultProvider{client: c, kvPath: kvPath, masterKey: mk}, nil
}

func NewVaultProviderWithClient(client *api.Client, kvPath string, masterKeyHex string) (*VaultProvider, error) {
	mk, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, err
	}
	if len(mk) != 32 {
		return nil, errors.New("master key must be 32 bytes (HEX)")
	}
	return &VaultProvider{client: client, kvPath: kvPath, masterKey: mk}, nil
}

func (p *VaultProvider) read(ctx context.Context) (map[string]interface{}, error) {
	secret, err := p.client.KVv2("secret").Get(ctx, p.kvPath)
	if err != nil {
		return nil, err
	}
	data := secret.Data
	if _, ok := data["current"]; !ok {
		if inner, ok2 := data["data"].(map[string]interface{}); ok2 {
			data = inner
		}
	}
	return data, nil
}

func (p *VaultProvider) write(ctx context.Context, data map[string]interface{}) error {
	_, err := p.client.KVv2("secret").Put(ctx, p.kvPath, data)
	return err
}

func (p *VaultProvider) GetCurrent(ctx context.Context) ([]byte, int, error) {
	data, err := p.read(ctx)
	if err != nil {
		// initialize
		kek := make([]byte, 32)
		if _, e := rand.Read(kek); e != nil {
			return nil, 0, e
		}
		nonce, ct, e := aead.Encrypt(p.masterKey, nil, kek)
		if e != nil {
			return nil, 0, e
		}
		init := map[string]interface{}{
			"current":  1,
			"versions": map[string]interface{}{"1": map[string]interface{}{"ciphertext": hex.EncodeToString(ct), "nonce": hex.EncodeToString(nonce)}},
		}
		if e := p.write(ctx, init); e != nil {
			return nil, 0, e
		}
		return kek, 1, nil
	}
	var cur int
	switch v := data["current"].(type) {
	case int:
		cur = v
	case float64:
		cur = int(v)
	case json.Number:
		if n, e := v.Int64(); e == nil {
			cur = int(n)
		}
	case string:
		if n, e := strconv.Atoi(v); e == nil {
			cur = n
		}
	}
	if cur == 0 {
		// treat as not initialized
		kek := make([]byte, 32)
		if _, e := rand.Read(kek); e != nil {
			return nil, 0, e
		}
		nonce, ct, e := aead.Encrypt(p.masterKey, nil, kek)
		if e != nil {
			return nil, 0, e
		}
		init := map[string]interface{}{
			"current":  1,
			"versions": map[string]interface{}{"1": map[string]interface{}{"ciphertext": hex.EncodeToString(ct), "nonce": hex.EncodeToString(nonce)}},
		}
		if e := p.write(ctx, init); e != nil {
			return nil, 0, e
		}
		return kek, 1, nil
	}
	versions, ok := data["versions"].(map[string]interface{})
	if !ok {
		// treat as not initialized
		kek := make([]byte, 32)
		if _, e := rand.Read(kek); e != nil {
			return nil, 0, e
		}
		nonce, ct, e := aead.Encrypt(p.masterKey, nil, kek)
		if e != nil {
			return nil, 0, e
		}
		init := map[string]interface{}{
			"current":  1,
			"versions": map[string]interface{}{"1": map[string]interface{}{"ciphertext": hex.EncodeToString(ct), "nonce": hex.EncodeToString(nonce)}},
		}
		if e := p.write(ctx, init); e != nil {
			return nil, 0, e
		}
		return kek, 1, nil
	}
	entry, ok := versions[fmt.Sprintf("%d", cur)].(map[string]interface{})
	if !ok {
		// treat as not initialized for this version and reset to 1
		kek := make([]byte, 32)
		if _, e := rand.Read(kek); e != nil {
			return nil, 0, e
		}
		nonce, ct, e := aead.Encrypt(p.masterKey, nil, kek)
		if e != nil {
			return nil, 0, e
		}
		init := map[string]interface{}{
			"current":  1,
			"versions": map[string]interface{}{"1": map[string]interface{}{"ciphertext": hex.EncodeToString(ct), "nonce": hex.EncodeToString(nonce)}},
		}
		if e := p.write(ctx, init); e != nil {
			return nil, 0, e
		}
		return kek, 1, nil
	}
	hexCt, _ := entry["ciphertext"].(string)
	nonceHex, _ := entry["nonce"].(string)
	ct, err := hex.DecodeString(hexCt)
	if err != nil {
		return nil, 0, err
	}
	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, 0, err
	}
	kek, err := aead.Decrypt(p.masterKey, nil, nonce, ct)
	if err != nil {
		return nil, 0, err
	}
	return kek, cur, nil
}

func (p *VaultProvider) GetByVersion(ctx context.Context, version int) ([]byte, error) {
	data, err := p.read(ctx)
	if err != nil {
		return nil, err
	}
	versions, ok := data["versions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("versions missing")
	}
	entry, ok := versions[fmt.Sprintf("%d", version)].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("version entry missing")
	}
	hexCt, _ := entry["ciphertext"].(string)
	nonceHex, _ := entry["nonce"].(string)
	ct, err := hex.DecodeString(hexCt)
	if err != nil {
		return nil, err
	}
	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, err
	}
	kek, err := aead.Decrypt(p.masterKey, nil, nonce, ct)
	if err != nil {
		return nil, err
	}
	return kek, nil
}

func (p *VaultProvider) Rotate(ctx context.Context) ([]byte, int, error) {
	data, err := p.read(ctx)
	if err != nil {
		return nil, 0, err
	}
	cur := 0
	switch v := data["current"].(type) {
	case int:
		cur = v
	case float64:
		cur = int(v)
	case json.Number:
		if n, e := v.Int64(); e == nil {
			cur = int(n)
		}
	case string:
		if n, e := strconv.Atoi(v); e == nil {
			cur = n
		}
	}
	next := cur + 1
	kek := make([]byte, 32)
	if _, err := rand.Read(kek); err != nil {
		return nil, 0, err
	}
	nonce, ct, err := aead.Encrypt(p.masterKey, nil, kek)
	if err != nil {
		return nil, 0, err
	}
	versions, ok := data["versions"].(map[string]interface{})
	if !ok {
		versions = map[string]interface{}{}
	}
	versions[fmt.Sprintf("%d", next)] = map[string]interface{}{"ciphertext": hex.EncodeToString(ct), "nonce": hex.EncodeToString(nonce)}
	data["versions"] = versions
	data["current"] = next
	if err := p.write(ctx, data); err != nil {
		return nil, 0, err
	}
	return kek, next, nil
}
