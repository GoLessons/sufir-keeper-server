package shutdown

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestGracefulShutdown_RunInvokesStartAndShutdown(t *testing.T) {
	logger := zap.NewNop()
	gs := NewGracefulShutdown(logger, 1*time.Second)

	startCalled := make(chan struct{}, 1)
	shutdownCalled := make(chan struct{}, 1)

	go func() {
		time.Sleep(50 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGINT)
	}()

	gs.Run(func() {
		startCalled <- struct{}{}
	}, func(_ context.Context) error {
		shutdownCalled <- struct{}{}
		return nil
	})

	select {
	case <-startCalled:
	default:
		t.Fatalf("start not invoked")
	}
	select {
	case <-shutdownCalled:
	default:
		t.Fatalf("shutdown not invoked")
	}
}

func TestGracefulShutdown_ShutdownErrorIsLogged(_ *testing.T) {
	logger := zap.NewExample()
	gs := NewGracefulShutdown(logger, 1*time.Second)

	go func() {
		time.Sleep(50 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGTERM)
	}()

	gs.Run(func() {}, func(_ context.Context) error {
		return errors.New("oops")
	})
}
