package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
)

func TestContentTypeValidationMiddleware_JSONRequired(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	final := ContentTypeValidationMiddleware()(h)

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
	var e api.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &e)
	if e.Message == nil || *e.Message != "content type must be application/json" {
		t.Fatalf("unexpected message: %v", e.Message)
	}
}

func TestContentTypeValidationMiddleware_FilesMultipartRequired(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	final := ContentTypeValidationMiddleware()(h)

	req := httptest.NewRequest(http.MethodPost, "/files/upload", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
	var e api.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &e)
	if e.Message == nil || *e.Message != "content type must be multipart/form-data" {
		t.Fatalf("unexpected message: %v", e.Message)
	}
}

func TestContentTypeValidationMiddleware_PassesValidTypes(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	final := ContentTypeValidationMiddleware()(h)

	req1 := httptest.NewRequest(http.MethodPost, "/items", nil)
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	final.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/files/upload", nil)
	req2.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	rr2 := httptest.NewRecorder()
	final.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}
}
