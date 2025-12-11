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
	}{
		{name: "ok-postgres", dsn: "postgres://u:p@h:5432/db", expect: nil},
		{name: "ok-postgresql", dsn: "postgresql://u:p@h:5432/db", expect: nil},
		{name: "empty", dsn: "", expect: errors.New("")},
		{name: "bad-scheme", dsn: "mysql://u:p@h:5432/db", expect: errors.New("")},
		{name: "missing-host", dsn: "postgres:///db", expect: errors.New("")},
		{name: "missing-db", dsn: "postgres://h", expect: errors.New("")},
	}
	for _, c := range cases {
		err := validateApplicationConfiguration(AppConfig{
			DB: DatabaseConfig{DataSourceName: c.dsn},
			S3: S3Config{Bucket: "mybucket"},
		})
		if c.expect == nil && err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if c.expect != nil && err == nil {
			t.Fatalf("%s: expected error, got nil", c.name)
		}
	}
}
