package config

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	"github.com/GoLessons/sufir-keeper-server/internal/db"
)

type ApplicationContainer struct {
	router         *chi.Mux
	logger         *zap.Logger
	databaseClient *db.Client
	httpServer     *http.Server
	configuration  AppConfig
}

func Initialize(parentContext context.Context) (*ApplicationContainer, error) {
	logger, err := createApplicationLogger()
	if err != nil {
		return nil, err
	}

	configuration, err := LoadApplicationConfiguration()
	if err != nil {
		_ = logger.Sync()
		return nil, err
	}

	databaseClient, err := createDatabaseClient(parentContext, configuration)
	if err != nil {
		_ = logger.Sync()
		return nil, err
	}

	router, httpServer := createHTTPServerAndRouter(configuration)
	tokenAuth, err := createJWTAuth(configuration)
	if err != nil {
		_ = logger.Sync()
		return nil, err
	}
	options := createChiServerOptions(router, logger, configuration, tokenAuth)
	tmpContainer := &ApplicationContainer{
		logger:         logger,
		configuration:  configuration,
		databaseClient: databaseClient,
		router:         router,
		httpServer:     httpServer,
	}
	serverImpl := createServerImplementation(tmpContainer, tokenAuth)
	handler := api.Handler(serverImpl, options)
	authVerify(router)
	if configuration.Server.MaxBodyBytes > 0 {
		httpServer.Handler = http.MaxBytesHandler(handler, configuration.Server.MaxBodyBytes)
	} else {
		httpServer.Handler = handler
	}
	return &ApplicationContainer{
		logger:         logger,
		configuration:  configuration,
		databaseClient: databaseClient,
		router:         router,
		httpServer:     httpServer,
	}, nil
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

func (container *ApplicationContainer) Close() error {
	if container == nil {
		return nil
	}

	var errs error
	if container.databaseClient != nil {
		if err := container.databaseClient.Close(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	if container.logger != nil {
		if err := container.logger.Sync(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}

func (container *ApplicationContainer) Router() *chi.Mux           { return container.router }
func (container *ApplicationContainer) Logger() *zap.Logger        { return container.logger }
func (container *ApplicationContainer) DatabaseClient() *db.Client { return container.databaseClient }
func (container *ApplicationContainer) HTTPServer() *http.Server   { return container.httpServer }
func (container *ApplicationContainer) Configuration() AppConfig   { return container.configuration }
