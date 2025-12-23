package httputil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
)

func TestDecodeJSON_Success(t *testing.T) {
	bodyBytes := []byte(`{"a":1}`)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(bodyBytes))
	var dst struct {
		A int `json:"a"`
	}
	err := DecodeJSON(req, &dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.A != 1 {
		t.Fatalf("unexpected value: %d", dst.A)
	}
}

func TestDecodeJSON_UnknownField(t *testing.T) {
	bodyBytes := []byte(`{"a":1,"extra":2}`)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(bodyBytes))
	var dst struct {
		A int `json:"a"`
	}
	err := DecodeJSON(req, &dst)
	if err == nil {
		t.Fatalf("expected error for unknown field, got nil")
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"key": "value"}
	WriteJSON(rec, http.StatusCreated, payload)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got["key"] != "value" {
		t.Fatalf("unexpected body: %v", got)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "bad_request", "message")
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var got apitypes.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Code == nil || *got.Code != http.StatusBadRequest {
		t.Fatalf("unexpected code: %v", got.Code)
	}
	if got.Error == nil || *got.Error != "bad_request" {
		t.Fatalf("unexpected error: %v", got.Error)
	}
	if got.Message == nil || *got.Message != "message" {
		t.Fatalf("unexpected message: %v", got.Message)
	}
}
