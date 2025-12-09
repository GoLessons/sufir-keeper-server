package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"go.uber.org/zap"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	fileshandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/files"
	"github.com/GoLessons/sufir-keeper-server/internal/app/middleware"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
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
		middleware.LoggingMiddleware(logger, middleware.HTTPLogLevels{Success: strings.TrimSpace(configuration.Log.LevelSuccess), ClientError: strings.TrimSpace(configuration.Log.LevelClientError), ServerError: strings.TrimSpace(configuration.Log.LevelServerError)}),
	}
	protected := middleware.AuthRequiredMiddleware(tokenAuth)
	jsonOnly := middleware.ContentTypeValidationMiddleware()
	middlewares := map[string][]api.MiddlewareFunc{
		"common":             common,
		"DELETE /auth":       {protected},
		"POST /auth":         {jsonOnly},
		"PATCH /auth":        {jsonOnly},
		"POST /register":     {jsonOnly},
		"POST /items":        {protected, jsonOnly},
		"PUT /items/{id}":    {protected, jsonOnly},
		"GET /items":         {protected, jsonOnly},
		"GET /items/{id}":    {protected, jsonOnly},
		"DELETE /items/{id}": {protected},
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

	s3Cfg := container.configuration.S3
	if strings.TrimSpace(s3Cfg.Endpoint) != "" && strings.TrimSpace(s3Cfg.AccessKey) != "" && strings.TrimSpace(s3Cfg.SecretKey) != "" && strings.TrimSpace(s3Cfg.Bucket) != "" {
        if client, err := s3.NewClient(strings.TrimSpace(s3Cfg.Endpoint), strings.TrimSpace(s3Cfg.AccessKey), strings.TrimSpace(s3Cfg.SecretKey), strings.TrimSpace(s3Cfg.Bucket)); err == nil {
            _ = client.EnsureBucket(context.Background())
            _ = client.SetBucketWebhookCreatedEvents(context.Background())
            wh := fileshandler.NewWebhookHandler(repository.NewItemRepository(container.databaseClient), client, provider, strings.TrimSpace(os.Getenv("MINIO_WEBHOOK_SECRET")))
            container.router.Post("/files/webhook-minio", wh.Handle)
        }
	}
	authVerify(container.router)
	server := api.NewServer(deps)
	return server
}

func authVerify(router *chi.Mux) {
	verify := func(w http.ResponseWriter, r *http.Request) {
		_, claims, err := jwtauth.FromContext(r.Context())
		if err != nil || claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		t, _ := claims["typ"].(string)
		if strings.ToLower(strings.TrimSpace(t)) != "access" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sub, _ := claims["sub"].(string)
		if strings.TrimSpace(sub) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-User-Id", strings.TrimSpace(sub))
		w.WriteHeader(http.StatusNoContent)
	}

	router.Post("/auth-verify", verify)
	router.Get("/auth-verify", verify)
}

func createJWTAuth(configuration AppConfig) (*jwtauth.JWTAuth, error) {
	secret := strings.TrimSpace(configuration.Auth.JwtSecret)
	if secret == "" {
		return nil, fmt.Errorf("секрет для JWT не должен быть пустым")
	}
	return jwtauth.New("HS256", []byte(secret), nil), nil
}
