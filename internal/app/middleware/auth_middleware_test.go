package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/jwtauth/v5"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
)

func TestAuthRequiredMiddleware_RejectsRefreshToken(t *testing.T) {
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	final := AuthRequiredMiddleware(tokenAuth)(h)

	claims := map[string]interface{}{"typ": "refresh"}
	_, tokenString, _ := tokenAuth.Encode(claims)
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	var e apitypes.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &e)
	if e.Message == nil || *e.Message != "Access token required" {
		t.Fatalf("expected message 'Access token required', got %v", e.Message)
	}
}

func TestAuthRequiredMiddleware_AllowsAccessToken(t *testing.T) {
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	final := AuthRequiredMiddleware(tokenAuth)(h)

	claims := map[string]interface{}{"typ": "access"}
	_, tokenString, _ := tokenAuth.Encode(claims)
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()
	final.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
