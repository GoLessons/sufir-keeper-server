package config

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"go.uber.org/zap"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	"github.com/GoLessons/sufir-keeper-server/internal/app/middleware"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

func createApplicationLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func createDatabaseClient(parentContext context.Context, configuration AppConfig) (*db.Client, error) {
	client, err := db.NewClient(parentContext, configuration.DB.DataSourceName, db.Options{
		MaxOpenConns:    configuration.DB.MaxOpenConns,
		MaxIdleConns:    configuration.DB.MaxIdleConns,
		ConnMaxLifetime: time.Duration(configuration.DB.ConnMaxLifetimeSeconds) * time.Second,
		ConnMaxIdleTime: time.Duration(configuration.DB.ConnMaxIdleTimeSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func createHTTPServerAndRouter(configuration AppConfig) (*chi.Mux, *http.Server) {
	address := configuration.Server.Address
	if address == "" {
		address = "0.0.0.0"
	}

	port := configuration.Server.Port
	if port == 0 {
		port = 8080
	}

	router := chi.NewRouter()

	httpServer := &http.Server{Addr: fmt.Sprintf("%s:%d", address, port), Handler: router}
	if configuration.Server.ReadTimeoutSeconds > 0 {
		httpServer.ReadTimeout = time.Duration(configuration.Server.ReadTimeoutSeconds) * time.Second
	}
	if configuration.Server.ReadHeaderTimeoutSeconds > 0 {
		httpServer.ReadHeaderTimeout = time.Duration(configuration.Server.ReadHeaderTimeoutSeconds) * time.Second
	}
	if configuration.Server.WriteTimeoutSeconds > 0 {
		httpServer.WriteTimeout = time.Duration(configuration.Server.WriteTimeoutSeconds) * time.Second
	}
	if configuration.Server.IdleTimeoutSeconds > 0 {
		httpServer.IdleTimeout = time.Duration(configuration.Server.IdleTimeoutSeconds) * time.Second
	}

	return router, httpServer
}

func createChiServerOptions(router *chi.Mux, logger *zap.Logger, configuration AppConfig, tokenAuth *jwtauth.JWTAuth) api.ChiServerOptions {
	common := []api.MiddlewareFunc{
		middleware.RecoverMiddleware(),
		middleware.ContentTypeValidationMiddleware(),
		middleware.LoggingMiddleware(logger, middleware.HTTPLogLevels{Success: strings.TrimSpace(configuration.Log.LevelSuccess), ClientError: strings.TrimSpace(configuration.Log.LevelClientError), ServerError: strings.TrimSpace(configuration.Log.LevelServerError)}),
	}
	protected := []api.MiddlewareFunc{middleware.AuthRequiredMiddleware(tokenAuth)}
	middlewares := map[string][]api.MiddlewareFunc{
		"common":             common,
		"DELETE /auth":       protected,
		"GET /items":         protected,
		"POST /items":        protected,
		"GET /items/{id}":    protected,
		"PUT /items/{id}":    protected,
		"DELETE /items/{id}": protected,
	}
	return api.ChiServerOptions{BaseURL: "", BaseRouter: router, Middlewares: middlewares, ErrorHandlerFunc: api.DefaultErrorHandler}
}

func createServerImplementation(container *ApplicationContainer, tokenAuth *jwtauth.JWTAuth) api.ServerInterface {
	deps := api.ServerDependencies{
		Logger:                 container.logger,
		DatabaseClient:         container.databaseClient,
		TokenAuth:              tokenAuth,
		AccessTokenTTLSeconds:  container.configuration.Auth.AccessTokenTTLSeconds,
		RefreshTokenTTLSeconds: container.configuration.Auth.RefreshTokenTTLSeconds,
		UsersRepository:        repository.NewUserRepository(container.databaseClient),
		ItemsRepository:        repository.NewItemRepository(container.databaseClient),
	}
	var provider keyencrypt.Provider
	cryptoCfg := container.configuration.Crypto
	if strings.TrimSpace(cryptoCfg.VaultAddr) != "" && strings.TrimSpace(cryptoCfg.VaultToken) != "" && strings.TrimSpace(cryptoCfg.VaultKVPath) != "" && strings.TrimSpace(cryptoCfg.MasterKeyHex) != "" {
		vp, err := keyencrypt.NewVaultProvider(strings.TrimSpace(cryptoCfg.VaultAddr), strings.TrimSpace(cryptoCfg.VaultToken), strings.TrimSpace(cryptoCfg.VaultKVPath), strings.TrimSpace(cryptoCfg.MasterKeyHex))
		if err == nil {
			provider = vp
		}
	}
	if provider == nil {
		provider = &keyencrypt.StaticProvider{Key: make([]byte, 32), Version: 1}
	}
	deps.KEKProvider = provider
	server := api.NewServer(deps)
	return server
}

func createJWTAuth(configuration AppConfig) (*jwtauth.JWTAuth, error) {
	secret := strings.TrimSpace(configuration.Auth.JwtSecret)
	if secret == "" {
		return nil, fmt.Errorf("секрет для JWT не должен быть пустым")
	}
	return jwtauth.New("HS256", []byte(secret), nil), nil
}
