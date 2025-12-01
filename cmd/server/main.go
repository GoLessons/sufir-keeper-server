//go:generate go tool oapi-codegen -config ../../tools/oapi.yaml ../../docs/schema.yaml

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoLessons/sufir-keeper-server/internal/api"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	router := chi.NewRouter()

	serverImpl := api.Unimplemented{}
	handler := api.Handler(serverImpl, api.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: api.DefaultErrorHandler, Middlewares: map[string][]api.MiddlewareFunc{"common": {api.LoggingMiddleware(logger)}}})

	srv := &http.Server{Addr: ":8080", Handler: handler}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Ошибка при работе сервера", zap.Error(err))
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Ошибка при завершении работы сервера", zap.Error(err))
	}
}
