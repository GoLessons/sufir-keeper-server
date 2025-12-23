//go:generate oapi-codegen -config ../../tools/oapi.yaml ../../docs/schema.yaml
//go:generate oapi-codegen -config ../../tools/oapi-server.yaml ../../docs/schema.yaml

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/GoLessons/sufir-keeper-server/internal/app/shutdown"

	"github.com/GoLessons/sufir-keeper-server/internal/config"

	"go.uber.org/zap"
)

func main() {
	container, err := config.Initialize(context.Background())
	if err != nil {
		fmt.Printf("Ошибка инициализации приложения: %s", err.Error())
		os.Exit(1)
	}
	defer func() { _ = container.Close() }()

	app := shutdown.NewGracefulShutdown(container.Logger(), 30*time.Second)
	app.Run(
		func() {
			server := container.HTTPServer()
			logger := container.Logger()

			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Error("Ошибка при работе сервера", zap.Error(err))
			}
		},
		func(ctx context.Context) error {
			return container.HTTPServer().Shutdown(ctx)
		},
	)
}
