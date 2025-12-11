package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
)

const maxLogBodyBytes = 16 * 1024

var (
	sensitiveHeaderKeys = map[string]struct{}{
		"authorization": {},
		"cookie":        {},
		"set-cookie":    {},
		"x-api-key":     {},
	}
	sensitiveJSONKeys = map[string]struct{}{
		"password":      {},
		"card_number":   {},
		"cvv":           {},
		"access_token":  {},
		"refresh_token": {},
		"token":         {},
		"secret":        {},
	}
)

func init() {
	normalizedJSON := make(map[string]struct{}, len(sensitiveJSONKeys))
	for k := range sensitiveJSONKeys {
		normalizedJSON[strings.ToLower(k)] = struct{}{}
	}
	sensitiveJSONKeys = normalizedJSON

	normalizedHeaders := make(map[string]struct{}, len(sensitiveHeaderKeys))
	for k := range sensitiveHeaderKeys {
		normalizedHeaders[strings.ToLower(k)] = struct{}{}
	}
	sensitiveHeaderKeys = normalizedHeaders
}

const (
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

type HTTPLogLevels struct {
	Success     string
	ClientError string
	ServerError string
}

func defaultHTTPLogLevels() HTTPLogLevels {
	return HTTPLogLevels{Success: logLevelInfo, ClientError: logLevelWarn, ServerError: logLevelError}
}

type requestBodyCapture struct {
	underlying     io.ReadCloser
	buffer         bytes.Buffer
	limit          int
	truncated      bool
	captureEnabled bool
}

func (r *requestBodyCapture) Read(p []byte) (int, error) {
	n, err := r.underlying.Read(p)
	if r.captureEnabled && n > 0 {
		remaining := r.limit - r.buffer.Len()
		if remaining > 0 {
			if n <= remaining {
				_, _ = r.buffer.Write(p[:n])
			} else {
				_, _ = r.buffer.Write(p[:remaining])
				r.truncated = true
			}
		} else {
			r.truncated = true
		}
	}
	return n, err
}

func (r *requestBodyCapture) Close() error { return r.underlying.Close() }

type responseRecorder struct {
	http.ResponseWriter
	panicError     error
	headerSnapshot http.Header
	panicStack     []byte
	body           bytes.Buffer
	status         int
	contentLength  int
	limit          int
	bodyTruncated  bool
}

func (s *responseRecorder) WriteHeader(status int) {
	s.status = status
	s.headerSnapshot = copyHeader(s.Header())
	s.ResponseWriter.WriteHeader(status)
}

func (s *responseRecorder) Write(b []byte) (int, error) {
	if s.headerSnapshot == nil {
		s.headerSnapshot = copyHeader(s.Header())
	}
	n, err := s.ResponseWriter.Write(b)
	s.contentLength += n
	remaining := s.limit - s.body.Len()
	if remaining > 0 {
		if n <= remaining {
			_, _ = s.body.Write(b[:n])
		} else {
			_, _ = s.body.Write(b[:remaining])
			s.bodyTruncated = true
		}
	} else {
		s.bodyTruncated = true
	}
	return n, err
}

func LoggingMiddleware(logger *zap.Logger, levels HTTPLogLevels) api.MiddlewareFunc {
	if strings.TrimSpace(levels.Success) == "" || strings.TrimSpace(levels.ClientError) == "" || strings.TrimSpace(levels.ServerError) == "" {
		d := defaultHTTPLogLevels()
		if strings.TrimSpace(levels.Success) == "" {
			levels.Success = d.Success
		}
		if strings.TrimSpace(levels.ClientError) == "" {
			levels.ClientError = d.ClientError
		}
		if strings.TrimSpace(levels.ServerError) == "" {
			levels.ServerError = d.ServerError
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			requestContentType := r.Header.Get("Content-Type")
			requestBodyAllowed := isTextContentType(requestContentType) && !strings.HasPrefix(strings.ToLower(requestContentType), "multipart/form-data")
			requestCapture := &requestBodyCapture{underlying: r.Body, limit: maxLogBodyBytes, captureEnabled: requestBodyAllowed}
			r.Body = requestCapture
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK, limit: maxLogBodyBytes}
			next.ServeHTTP(recorder, r)
			duration := time.Since(startTime)
			statusCode := recorder.status
			if statusCode < 400 {
				logLevel := levels.Success
				fields := []zap.Field{zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Int("status", statusCode), zap.Int("bytes", recorder.contentLength), zap.Duration("duration", duration)}
				switch logLevel {
				case logLevelDebug:
					logger.Debug("http", fields...)
				case logLevelInfo:
					logger.Info("http", fields...)
				case logLevelWarn:
					logger.Warn("http", fields...)
				case logLevelError:
					logger.Error("http", fields...)
				default:
					logger.Info("http", fields...)
				}
				return
			}
			maskedRequestHeaders := maskHeaders(r.Header)
			responseHeaders := recorder.headerSnapshot
			if responseHeaders == nil {
				responseHeaders = copyHeader(recorder.Header())
			}
			maskedResponseHeaders := maskHeaders(responseHeaders)
			var requestBodyValue string
			var responseBodyValue string
			requestBodyTruncated := false
			responseBodyTruncated := false
			if requestBodyAllowed {
				requestBytes := requestCapture.buffer.Bytes()
				requestBodyTruncated = requestCapture.truncated
				if strings.Contains(strings.ToLower(requestContentType), "application/json") {
					masked := maskJSON(requestBytes)
					requestBodyValue = string(masked)
				} else {
					requestBodyValue = string(requestBytes)
				}
			}
			responseContentType := strings.ToLower(responseHeaders.Get("Content-Type"))
			var responseBodyAllowed bool
			if strings.TrimSpace(responseContentType) == "" {
				responseBodyAllowed = true
			} else {
				responseBodyAllowed = isTextContentType(responseContentType) && !strings.HasPrefix(responseContentType, "multipart/form-data") && !isBinaryContentType(responseContentType)
			}
			if responseBodyAllowed {
				responseBytes := recorder.body.Bytes()
				responseBodyTruncated = recorder.bodyTruncated
				if strings.Contains(responseContentType, "application/json") {
					masked := maskJSON(responseBytes)
					responseBodyValue = string(masked)
				} else {
					responseBodyValue = string(responseBytes)
				}
			}
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", statusCode),
				zap.Int("bytes", recorder.contentLength),
				zap.Duration("duration", duration),
				zap.Any("request_headers", maskedRequestHeaders),
				zap.Any("response_headers", maskedResponseHeaders),
				zap.String("request_body", requestBodyValue),
				zap.String("response_body", responseBodyValue),
				zap.Bool("request_truncated", requestBodyTruncated),
				zap.Bool("response_truncated", responseBodyTruncated),
			}
			if statusCode >= 500 && recorder.panicError != nil {
				fields = append(fields, zap.String("error", recorder.panicError.Error()))
				fields = append(fields, zap.ByteString("stack", recorder.panicStack))
			}
			if statusCode >= 500 {
				switch levels.ServerError {
				case logLevelDebug:
					logger.Debug("http", fields...)
				case logLevelInfo:
					logger.Info("http", fields...)
				case logLevelWarn:
					logger.Warn("http", fields...)
				case logLevelError:
					logger.Error("http", fields...)
				default:
					logger.Error("http", fields...)
				}
			} else {
				switch levels.ClientError {
				case logLevelDebug:
					logger.Debug("http", fields...)
				case logLevelInfo:
					logger.Info("http", fields...)
				case logLevelWarn:
					logger.Warn("http", fields...)
				case logLevelError:
					logger.Error("http", fields...)
				default:
					logger.Warn("http", fields...)
				}
			}
		})
	}
}

func maskHeaders(headers http.Header) map[string][]string {
	if headers == nil {
		return map[string][]string{}
	}
	result := make(map[string][]string, len(headers))
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if _, ok := sensitiveHeaderKeys[lowerKey]; ok {
			maskedValues := make([]string, len(values))
			for i := range values {
				maskedValues[i] = "***"
			}
			result[key] = maskedValues
		} else {
			copiedValues := make([]string, len(values))
			copy(copiedValues, values)
			result[key] = copiedValues
		}
	}
	return result
}

func isTextContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	if ct == "" {
		return false
	}
	return strings.HasPrefix(ct, "text/") || strings.Contains(ct, "application/json") || strings.Contains(ct, "application/x-www-form-urlencoded")
}

func isBinaryContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	if ct == "" {
		return false
	}
	if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/") {
		return true
	}
	if strings.Contains(ct, "application/octet-stream") || strings.Contains(ct, "application/zip") || strings.Contains(ct, "application/pdf") {
		return true
	}
	return false
}

func copyHeader(h http.Header) http.Header {
	if h == nil {
		return http.Header{}
	}
	c := make(http.Header, len(h))
	for k, vv := range h {
		nv := make([]string, len(vv))
		copy(nv, vv)
		c[k] = nv
	}
	return c
}

func maskJSON(data []byte) []byte {
	var v interface{}
	if json.Unmarshal(data, &v) != nil {
		return data
	}
	masked := maskJSONValue(v)
	encoded, err := json.Marshal(masked)
	if err != nil {
		return data
	}
	return encoded
}

func maskJSONValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		res := make(map[string]interface{}, len(t))
		for k, val := range t {
			if _, ok := sensitiveJSONKeys[strings.ToLower(k)]; ok {
				res[k] = "***"
			} else {
				res[k] = maskJSONValue(val)
			}
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(t))
		for i := range t {
			res[i] = maskJSONValue(t[i])
		}
		return res
	default:
		return t
	}
}
