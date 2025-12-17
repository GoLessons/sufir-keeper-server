package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	model "github.com/GoLessons/sufir-keeper-server/internal/api/types"
)

func TestDefaultErrorHandler_Types(t *testing.T) {
	cases := []struct {
		err  func() error
		code string
	}{
		{err: func() error { return &RequiredParamError{ParamName: "p"} }, code: "required_param"},
		{err: func() error { return &RequiredHeaderError{ParamName: "h"} }, code: "required_header"},
		{err: func() error {
			return &UnmarshalingParamError{ParamName: "b", Err: json.Unmarshal([]byte("x"), &struct{}{})}
		}, code: "invalid_json"},
		{err: func() error {
			return &InvalidParamFormatError{ParamName: "f", Err: json.Unmarshal([]byte("x"), &struct{}{})}
		}, code: "invalid_format"},
		{err: func() error { return &TooManyValuesForParamError{ParamName: "t", Count: 2} }, code: "too_many_values"},
		{err: func() error { return &UnescapedCookieParamError{ParamName: "c"} }, code: "invalid_cookie"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		DefaultErrorHandler(rec, httptest.NewRequest(http.MethodGet, "/x", nil), c.err())
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("unexpected content-type: %s", ct)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		var got model.Error
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if got.Error == nil || *got.Error != c.code {
			t.Fatalf("unexpected error field: %v", got.Error)
		}
		if got.Code == nil || *got.Code != http.StatusBadRequest {
			t.Fatalf("unexpected code: %v", got.Code)
		}
		if got.Message == nil || *got.Message == "" {
			t.Fatalf("empty message")
		}
	}
}
