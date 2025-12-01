package api

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type statusRecorder struct {
	http.ResponseWriter
	status        int
	contentLength int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.contentLength += n
	return n, err
}

func LoggingMiddleware(logger *zap.Logger) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			duration := time.Since(start)
			logger.Info("http", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Int("status", rec.status), zap.Int("bytes", rec.contentLength), zap.Duration("duration", duration))
		})
	}
}
