package config

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	"github.com/GoLessons/sufir-keeper-server/internal/db"
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

func createChiServerOptions(router *chi.Mux, logger *zap.Logger, configuration AppConfig) api.ChiServerOptions {
	return api.ChiServerOptions{
		BaseURL:          "",
		BaseRouter:       router,
		Middlewares:      map[string][]api.MiddlewareFunc{"common": {api.RecoverMiddleware(), api.ContentTypeValidationMiddleware(), api.LoggingMiddleware(logger, api.HTTPLogLevels{Success: strings.TrimSpace(configuration.Log.LevelSuccess), ClientError: strings.TrimSpace(configuration.Log.LevelClientError), ServerError: strings.TrimSpace(configuration.Log.LevelServerError)})}},
		ErrorHandlerFunc: api.DefaultErrorHandler,
	}
}

func createServerImplementation(_ *ApplicationContainer) api.ServerInterface {
	return api.Unimplemented{}
}
