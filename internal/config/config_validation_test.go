package config

import "testing"

func TestValidateApplicationConfigurationErrors(t *testing.T) {
	cfg := AppConfig{}
	cfg.DB.DataSourceName = ""
	cfg.S3.Bucket = ""
	if err := validateApplicationConfiguration(cfg); err == nil {
		t.Fatalf("expected error for empty dsn")
	}

	cfg.DB.DataSourceName = "http://host/db"
	cfg.S3.Bucket = ""
	if err := validateApplicationConfiguration(cfg); err == nil {
		t.Fatalf("expected error for wrong scheme")
	}

	cfg.DB.DataSourceName = "postgres://:5432/db"
	cfg.S3.Bucket = ""
	if err := validateApplicationConfiguration(cfg); err == nil {
		t.Fatalf("expected error for missing host")
	}

	cfg.DB.DataSourceName = "postgres://localhost:5432/"
	cfg.S3.Bucket = ""
	if err := validateApplicationConfiguration(cfg); err == nil {
		t.Fatalf("expected error for missing database name")
	}

	cfg.DB.DataSourceName = "postgres://localhost:5432/db"
	cfg.S3.Bucket = ""
	if err := validateApplicationConfiguration(cfg); err == nil {
		t.Fatalf("expected error for empty bucket")
	}
}

func TestValidateApplicationConfigurationSuccess(t *testing.T) {
	cfg := AppConfig{}
	cfg.DB.DataSourceName = "postgres://localhost:5432/mydb"
	cfg.S3.Bucket = "bucket"
	if err := validateApplicationConfiguration(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
