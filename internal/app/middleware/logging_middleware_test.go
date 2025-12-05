package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newJSONLogger() (*zap.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	encCfg := zap.NewProductionEncoderConfig()
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(buf), zapcore.DebugLevel)
	logger := zap.New(core)
	return logger, buf
}

func parseLastLog(buf *bytes.Buffer) map[string]interface{} {
	data := buf.Bytes()
	idx := bytes.LastIndexByte(data, '\n')
	if idx == -1 {
		idx = len(data)
	}
	line := data[:idx]
	var m map[string]interface{}
	_ = json.Unmarshal(line, &m)
	return m
}

func TestLoggingMiddleware_InfoOn2xx(t *testing.T) {
	logger, buf := newJSONLogger()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = b
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"login":"user","password":"secret1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer abcdefgh")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	const levelInfo = "info"
	if log["level"] != levelInfo {
		t.Fatalf("expected info level, got %v", log["level"])
	}
	if _, ok := log["request_body"]; ok {
		t.Fatalf("2xx should not contain request_body")
	}
	if _, ok := log["response_body"]; ok {
		t.Fatalf("2xx should not contain response_body")
	}
	if v, ok := log["status"].(float64); !ok || int(v) != 200 {
		t.Fatalf("expected status 200, got %v", log["status"])
	}
}

func TestLoggingMiddleware_WarnOn4xx_WithMaskedBodies(t *testing.T) {
	logger, buf := newJSONLogger()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad","password":"secret2"}`))
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodPost, "/bad", strings.NewReader(`{"login":"user","password":"secret1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer abcdefgh")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	const levelWarn = "warn"
	if log["level"] != levelWarn {
		t.Fatalf("expected warn level, got %v", log["level"])
	}
	rb, _ := log["request_body"].(string)
	if rb == "" {
		t.Fatalf("expected non-empty request_body")
	}
	if !strings.Contains(rb, "password") {
		t.Fatalf("request_body should contain masked password field")
	}
	if strings.Contains(rb, "secret1") {
		t.Fatalf("password value should be masked")
	}
	respb, _ := log["response_body"].(string)
	if respb == "" {
		t.Fatalf("expected non-empty response_body")
	}
	if strings.Contains(respb, "secret2") {
		t.Fatalf("response password should be masked")
	}
	hdrs, _ := log["request_headers"].(map[string]interface{})
	if hdrs == nil {
		t.Fatalf("expected request_headers present")
	}
}

func TestLoggingMiddleware_ErrorOn5xx_PanicWithStack(t *testing.T) {
	logger, buf := newJSONLogger()
	h := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(RecoverMiddleware()(h))
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	const levelError = "error"
	if log["level"] != levelError {
		t.Fatalf("expected error level, got %v", log["level"])
	}
	if v, ok := log["status"].(float64); !ok || int(v) != 500 {
		t.Fatalf("expected status 500, got %v", log["status"])
	}
	if _, ok := log["error"].(string); !ok {
		t.Fatalf("expected error field present")
	}
	if _, ok := log["stack"].(string); !ok {
		t.Fatalf("expected stack field present")
	}
}

func TestLoggingMiddleware_BinaryResponse_NoBody(t *testing.T) {
	logger, buf := newJSONLogger()
	payload := bytes.Repeat([]byte{0x01}, 1024)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(payload)
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodGet, "/bin", nil)
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	const levelError = "error"
	if log["level"] != levelError {
		t.Fatalf("expected error level, got %v", log["level"])
	}
	if v, ok := log["status"].(float64); !ok || int(v) != 500 {
		t.Fatalf("expected status 500, got %v", log["status"])
	}
	if v, ok := log["response_body"].(string); !ok || v != "" {
		t.Fatalf("expected empty response_body for binary content")
	}
}

func TestLoggingMiddleware_RequestTruncatedFlag(t *testing.T) {
	logger, buf := newJSONLogger()
	long := strings.Repeat("a", maxLogBodyBytes+1024)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("short"))
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodPost, "/trim", strings.NewReader(long))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	if log["level"] != "warn" {
		t.Fatalf("expected warn level, got %v", log["level"])
	}
	rb, _ := log["request_body"].(string)
	if len(rb) != maxLogBodyBytes {
		t.Fatalf("expected request_body length %d, got %d", maxLogBodyBytes, len(rb))
	}
	if v, ok := log["request_truncated"].(bool); !ok || !v {
		t.Fatalf("expected request_truncated=true")
	}
}

func TestLoggingMiddleware_MultipartRequest_NotLoggedBody(t *testing.T) {
	logger, buf := newJSONLogger()
	body := "--boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.txt\"\r\nContent-Type: text/plain\r\n\r\ncontent\r\n--boundary--\r\n"
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodPost, "/multipart", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	if log["level"] != "warn" {
		t.Fatalf("expected warn level, got %v", log["level"])
	}
	if v, ok := log["request_body"].(string); !ok || v != "" {
		t.Fatalf("expected empty request_body for multipart")
	}
	if v, ok := log["response_body"].(string); !ok || v == "" {
		t.Fatalf("expected non-empty response_body")
	}
}

func TestLoggingMiddleware_ResponseTruncatedFlag(t *testing.T) {
	logger, buf := newJSONLogger()
	long := strings.Repeat("b", maxLogBodyBytes+2048)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(long))
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodGet, "/resp-trim", nil)
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	if log["level"] != "error" {
		t.Fatalf("expected error level, got %v", log["level"])
	}
	rb, _ := log["response_body"].(string)
	if len(rb) != maxLogBodyBytes {
		t.Fatalf("expected response_body length %d, got %d", maxLogBodyBytes, len(rb))
	}
	if v, ok := log["response_truncated"].(bool); !ok || !v {
		t.Fatalf("expected response_truncated=true")
	}
}

func TestLoggingMiddleware_MasksAuthorizationHeader(t *testing.T) {
	logger, buf := newJSONLogger()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	})
	final := LoggingMiddleware(logger, defaultHTTPLogLevels())(h)
	req := httptest.NewRequest(http.MethodGet, "/hdr", nil)
	req.Header.Set("Authorization", "Bearer verylongsecretvalue")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)
	log := parseLastLog(buf)
	const levelWarn = "warn"
	if log["level"] != levelWarn {
		t.Fatalf("expected warn level, got %v", log["level"])
	}
	hdrs, _ := log["request_headers"].(map[string]interface{})
	if hdrs == nil {
		t.Fatalf("expected request_headers present")
	}
	authValues, _ := hdrs["Authorization"].([]interface{})
	if len(authValues) == 0 {
		t.Fatalf("expected Authorization header present")
	}
	masked, _ := authValues[0].(string)
	if masked == "Bearer verylongsecretvalue" {
		t.Fatalf("expected masked Authorization header, got original")
	}
}
