package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type GracefulShutdown struct {
	logger  *zap.Logger
	timeout time.Duration
}

func NewGracefulShutdown(logger *zap.Logger, timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		logger:  logger,
		timeout: timeout,
	}
}

func (gs *GracefulShutdown) Run(start func(), shutdown func(context.Context) error) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		start()
	}()

	<-stop
	signal.Stop(stop)
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), gs.timeout)
	defer cancel()

	err := shutdown(ctx)
	if err != nil {
		gs.logger.Error("Ошибка при завершении работы", zap.Error(err))
	}
}
