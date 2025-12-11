package middleware

import (
	"encoding/json"
	"mime"
	"net/http"
	"strings"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	model "github.com/GoLessons/sufir-keeper-server/internal/api/types"
)

func RequireJSONMiddleware() api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := strings.ToUpper(r.Method)
			if m == http.MethodPost || m == http.MethodPut || m == http.MethodPatch {
				ct := strings.TrimSpace(r.Header.Get("Content-Type"))
				mt, _, err := mime.ParseMediaType(ct)
				if err != nil || mt != "application/json" {
					writeUnsupportedMediaType(w, "application/json")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireMultipartFormDataMiddleware() api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := strings.ToUpper(r.Method)
			if m == http.MethodPost || m == http.MethodPut || m == http.MethodPatch {
				ct := strings.TrimSpace(r.Header.Get("Content-Type"))
				mt, _, err := mime.ParseMediaType(ct)
				if err != nil || mt != "multipart/form-data" {
					writeUnsupportedMediaType(w, "multipart/form-data")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeUnsupportedMediaType(w http.ResponseWriter, expected string) {
	status := http.StatusUnsupportedMediaType
	code := "unsupported_media_type"
	message := "content type must be " + expected
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.Error{Code: &status, Error: &code, Message: &message})
}
