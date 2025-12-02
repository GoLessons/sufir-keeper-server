package api

import (
    "encoding/json"
    "net/http"
    "strings"
)

func ContentTypeValidationMiddleware() MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            method := strings.ToUpper(r.Method)
            if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
                ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
                if strings.HasPrefix(r.URL.Path, "/files") {
                    if ct == "" || !strings.HasPrefix(ct, "multipart/form-data") {
                        writeUnsupportedMediaType(w, "multipart/form-data")
                        return
                    }
                } else {
                    if ct == "" || !strings.Contains(ct, "application/json") {
                        writeUnsupportedMediaType(w, "application/json")
                        return
                    }
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
    _ = json.NewEncoder(w).Encode(Error{Code: &status, Error: &code, Message: &message})
}
