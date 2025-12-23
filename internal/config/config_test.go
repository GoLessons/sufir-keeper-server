package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadApplicationConfigurationJsonOnly(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")
	content := []byte(`{"db":{"dsn":"postgres://u:p@h:5432/db?sslmode=disable"},"s3":{"bucket":"mybucket"}}`)
	if err := os.WriteFile(jsonPath, content, 0o644); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}
	os.Setenv("APP_CONFIG_PATH", jsonPath)
	defer os.Unsetenv("APP_CONFIG_PATH")

	cfg, err := LoadApplicationConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DB.DataSourceName != "postgres://u:p@h:5432/db?sslmode=disable" {
		t.Fatalf("unexpected dsn: %s", cfg.DB.DataSourceName)
	}
}

func TestValidateApplicationConfiguration(t *testing.T) {
	cases := []struct {
		expect error
		name   string
		dsn    string
		bucket string
	}{
		{name: "ok-postgres", dsn: "postgres://u:p@h:5432/db", bucket: "mybucket", expect: nil},
		{name: "ok-postgresql", dsn: "postgresql://u:p@h:5432/db", bucket: "mybucket", expect: nil},
		{name: "empty", dsn: "", bucket: "mybucket", expect: errors.New("")},
		{name: "bad-scheme", dsn: "mysql://u:p@h:5432/db", bucket: "mybucket", expect: errors.New("")},
		{name: "missing-host", dsn: "postgres:///db", bucket: "mybucket", expect: errors.New("")},
		{name: "missing-db", dsn: "postgres://h", bucket: "mybucket", expect: errors.New("")},
		{name: "missing-bucket", dsn: "postgres://u:p@h:5432/db", bucket: "", expect: errors.New("")},
	}
	for _, c := range cases {
		err := validateApplicationConfiguration(AppConfig{
			DB: DatabaseConfig{DataSourceName: c.dsn},
			S3: S3Config{Bucket: c.bucket},
		})
		if c.expect == nil && err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if c.expect != nil && err == nil {
			t.Fatalf("%s: expected error, got nil", c.name)
		}
	}
}

func TestLoadApplicationConfigurationDefaults(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")
	content := []byte(`{"db":{"dsn":"postgres://u:p@h:5432/db"},"s3":{"bucket":"bucket-name"}}`)
	if err := os.WriteFile(jsonPath, content, 0o644); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}

	keys := []string{
		"APP_CONFIG_PATH",
		"DB_DSN",
		"S3_BUCKET",
		"S3_BUCKET_PROTECTED",
		"HTTP_LOG_SUCCESS",
		"HTTP_LOG_4XX",
		"HTTP_LOG_5XX",
		"AUTH_ACCESS_TTL",
		"AUTH_REFRESH_TTL",
	}
	values := map[string]string{}
	present := map[string]bool{}
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if ok {
			present[key] = true
			values[key] = value
		} else {
			present[key] = false
		}
		os.Unsetenv(key)
	}
	defer func() {
		for _, key := range keys {
			if present[key] {
				os.Setenv(key, values[key])
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	if err := os.Setenv("APP_CONFIG_PATH", jsonPath); err != nil {
		t.Fatalf("failed to set APP_CONFIG_PATH: %v", err)
	}

	cfg, err := LoadApplicationConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Log.LevelSuccess != "info" {
		t.Fatalf("unexpected success level: %s", cfg.Log.LevelSuccess)
	}
	if cfg.Log.LevelClientError != "warn" {
		t.Fatalf("unexpected client error level: %s", cfg.Log.LevelClientError)
	}
	if cfg.Log.LevelServerError != "error" {
		t.Fatalf("unexpected server error level: %s", cfg.Log.LevelServerError)
	}
	if cfg.Auth.AccessTokenTTLSeconds != 3600 {
		t.Fatalf("unexpected access token ttl: %d", cfg.Auth.AccessTokenTTLSeconds)
	}
	if cfg.Auth.RefreshTokenTTLSeconds != 30*24*3600 {
		t.Fatalf("unexpected refresh token ttl: %d", cfg.Auth.RefreshTokenTTLSeconds)
	}
	if cfg.S3.BucketProtected != "bucket-name-protected" {
		t.Fatalf("unexpected protected bucket: %s", cfg.S3.BucketProtected)
	}
}
