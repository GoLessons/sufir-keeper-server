package config

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sufir/go-set-me-up/setup"
	setupenv "github.com/Sufir/go-set-me-up/setup/source/env"
	setupflags "github.com/Sufir/go-set-me-up/setup/source/flags"
	jsonfile "github.com/Sufir/go-set-me-up/setup/source/json-file"
)

type AppConfig struct {
	Log    LogConfig      `json:"log"`
	DB     DatabaseConfig `json:"db"`
	Server ServerConfig   `json:"server"`
}

type DatabaseConfig struct {
	DataSourceName         string `json:"dsn" env:"DB_DSN" flag:"db-dsn" flagShort:"d"`
	MaxOpenConns           int    `json:"max_open_conns" env:"DB_MAX_OPEN" flag:"db-max-open"`
	MaxIdleConns           int    `json:"max_idle_conns" env:"DB_MAX_IDLE" flag:"db-max-idle"`
	ConnMaxLifetimeSeconds int    `json:"conn_max_lifetime_seconds" env:"DB_LIFETIME" flag:"db-lifetime"`
	ConnMaxIdleTimeSeconds int    `json:"conn_max_idle_time_seconds" env:"DB_IDLE" flag:"db-idle"`
}

type ServerConfig struct {
	Address                  string `json:"address" env:"HTTP_ADDR" flag:"addr" flagShort:"a"`
	Port                     int    `json:"port" env:"HTTP_PORT" flag:"port" flagShort:"P"`
	ReadTimeoutSeconds       int    `json:"read_timeout_seconds" env:"HTTP_READ_TIMEOUT" flag:"read-timeout"`
	ReadHeaderTimeoutSeconds int    `json:"read_header_timeout_seconds" env:"HTTP_HEADER_TIMEOUT" flag:"header-timeout"`
	WriteTimeoutSeconds      int    `json:"write_timeout_seconds" env:"HTTP_WRITE_TIMEOUT" flag:"write-timeout"`
	IdleTimeoutSeconds       int    `json:"idle_timeout_seconds" env:"HTTP_IDLE_TIMEOUT" flag:"idle-timeout"`
	MaxBodyBytes             int64  `json:"max_body_bytes" env:"HTTP_MAX_BODY" flag:"max-body"`
}

type LogConfig struct {
	LevelSuccess     string `json:"level_success" env:"HTTP_LOG_SUCCESS" flag:"log-success"`
	LevelClientError string `json:"level_client_error" env:"HTTP_LOG_4XX" flag:"log-4xx"`
	LevelServerError string `json:"level_server_error" env:"HTTP_LOG_5XX" flag:"log-5xx"`
}

func LoadApplicationConfiguration() (AppConfig, error) {
	var configuration AppConfig
	var sources []setup.Source

	if p, ok := os.LookupEnv("APP_CONFIG_PATH"); ok {
		if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
			sources = append(sources, jsonfile.NewSource(p, setup.ModeOverride))
		}
	}

	executablePath, err := os.Executable()
	if err == nil {
		directory := filepath.Dir(executablePath)
		path := filepath.Join(directory, "config.json")
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			sources = append(sources, jsonfile.NewSource(path, setup.ModeOverride))
		}
	}

	sources = append(sources, setupenv.NewSource("", ",", setup.ModeOverride))
	sources = append(sources, setupflags.NewSource(setup.ModeOverride))

	loader := setup.NewLoader(sources...)
	if err := loader.Load(&configuration); err != nil {
		return AppConfig{}, err
	}
	if err := validateApplicationConfiguration(configuration); err != nil {
		return AppConfig{}, err
	}

	if strings.TrimSpace(configuration.Log.LevelSuccess) == "" {
		configuration.Log.LevelSuccess = "info"
	}
	if strings.TrimSpace(configuration.Log.LevelClientError) == "" {
		configuration.Log.LevelClientError = "warn"
	}
	if strings.TrimSpace(configuration.Log.LevelServerError) == "" {
		configuration.Log.LevelServerError = "error"
	}
	return configuration, nil
}

func validateApplicationConfiguration(configuration AppConfig) error {
	value := strings.TrimSpace(configuration.DB.DataSourceName)
	if value == "" {
		return errors.New("строка подключения к Postgres не должна быть пустой")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return errors.New("строка подключения к Postgres имеет неверный формат URL")
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return errors.New("схема URL должна быть postgres или postgresql")
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("в URL отсутствует хост")
	}

	database := strings.TrimPrefix(parsed.Path, "/")
	if strings.TrimSpace(database) == "" {
		return errors.New("в URL отсутствует имя базы данных")
	}

	return nil
}
