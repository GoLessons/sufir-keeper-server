package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestRouter_UploadFile_MissingHeader(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "required_header", resp["error"])
}

func TestRouter_DownloadFile_InvalidUUID(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/not-uuid", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "invalid_format", resp["error"])
}

func TestRouter_GetItems_InvalidQuery(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items?limit=abc", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "invalid_format", resp["error"])
}

func TestRouter_AuthVerifyRoutes(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/auth-verify", nil)
	h.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusNotImplemented, rec1.Code)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/auth-verify", nil)
	h.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusNotImplemented, rec2.Code)
}

func TestRouter_ItemsPathParamErrors(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPut} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/items/not-uuid", nil)
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
}

func TestRouter_RegisterUserRoute(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotImplemented, rec.Code)
}

func TestRouter_AuthRoutes(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	for _, method := range []string{http.MethodDelete, http.MethodPatch, http.MethodPost} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/auth", nil)
		h.ServeHTTP(rec, req)
		if method == http.MethodDelete || method == http.MethodPost {
			require.Equal(t, http.StatusNotImplemented, rec.Code)
		} else {
			require.Equal(t, http.StatusNotImplemented, rec.Code)
		}
	}
}

func TestRouter_PresignFileRoute(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files/presign", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotImplemented, rec.Code)
}

func TestRouter_UploadFile_TooManyHeaderValues(t *testing.T) {
	r := chi.NewRouter()
	h := Handler(Unimplemented{}, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: map[string][]MiddlewareFunc{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files", nil)
	req.Header.Add("X-File-ID", "id1")
	req.Header.Add("X-File-ID", "id2")
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "too_many_values", resp["error"])
}
