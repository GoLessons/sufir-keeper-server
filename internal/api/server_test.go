package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestServer_ServiceUnavailableWhenDepsNil(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	s.CreateItem(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("CreateItem expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/items/00000000-0000-0000-0000-000000000000", nil)
	s.GetItem(rec, req, [16]byte{})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GetItem expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/items/00000000-0000-0000-0000-000000000000", nil)
	s.UpdateItem(rec, req, [16]byte{})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("UpdateItem expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/files/presign", nil)
	s.PresignFile(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("PresignFile expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/files/00000000-0000-0000-0000-000000000000", nil)
	tok := jwt.New()
	_ = tok.Set("sub", "00000000-0000-0000-0000-000000000000")
	_ = tok.Set("typ", "access")
	req = req.WithContext(jwtauth.NewContext(req.Context(), tok, nil))
	s.DownloadFile(rec, req, [16]byte{})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("DownloadFile expected 503, got %d", rec.Code)
	}
}
