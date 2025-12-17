package config

import "testing"

func TestCreateHTTPServerAndRouter_Defaults(t *testing.T) {
	router, server := createHTTPServerAndRouter(AppConfig{Server: ServerConfig{}})
	if router == nil || server == nil {
		t.Fatalf("router/server nil")
	}
	if server.Addr == "" {
		t.Fatalf("server addr empty")
	}
}

func TestCreateJWTAuth(t *testing.T) {
	if _, err := createJWTAuth(AppConfig{Auth: AuthConfig{JwtSecret: ""}}); err == nil {
		t.Fatalf("expected error for empty secret")
	}
	if _, err := createJWTAuth(AppConfig{Auth: AuthConfig{JwtSecret: "x"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateChiServerOptions_Keys(t *testing.T) {
	opts := createChiServerOptions(nil, nil, AppConfig{Log: LogConfig{}, Server: ServerConfig{}}, nil)
	if len(opts.Middlewares["common"]) == 0 {
		t.Fatalf("common middlewares missing")
	}
	keys := []string{"DELETE /auth", "POST /auth", "PATCH /auth", "POST /register", "POST /items", "PUT /items/{id}", "GET /items", "GET /items/{id}", "DELETE /items/{id}", "POST /files", "POST /files/presign", "GET /files/{fileId}", "GET /auth-verify", "POST /auth-verify"}
	for _, k := range keys {
		if _, ok := opts.Middlewares[k]; !ok {
			t.Fatalf("middleware key missing: %s", k)
		}
	}
}

func TestCreateHTTPServerAndRouter_Timeouts(t *testing.T) {
	router, server := createHTTPServerAndRouter(AppConfig{Server: ServerConfig{
		ReadTimeoutSeconds:       1,
		ReadHeaderTimeoutSeconds: 1,
		WriteTimeoutSeconds:      1,
		IdleTimeoutSeconds:       1,
	}})
	if router == nil || server == nil {
		t.Fatalf("router/server nil")
	}
	// Ensure non-zero timeouts
	if server.ReadTimeout <= 0 || server.ReadHeaderTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("timeouts not set")
	}
}
