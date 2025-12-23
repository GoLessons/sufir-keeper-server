package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestCreateHTTPServerAndRouterDefaults(t *testing.T) {
	router, srv := createHTTPServerAndRouter(AppConfig{})
	if router == nil || srv == nil {
		t.Fatalf("router or server is nil")
	}
	if srv.Addr != "0.0.0.0:8080" {
		t.Fatalf("unexpected addr: %s", srv.Addr)
	}
	if srv.ReadTimeout != 0 || srv.ReadHeaderTimeout != 0 || srv.WriteTimeout != 0 || srv.IdleTimeout != 0 {
		t.Fatalf("unexpected non-zero timeouts")
	}
}

func TestCreateHTTPServerAndRouterWithTimeouts(t *testing.T) {
	cfg := AppConfig{
		Server: ServerConfig{
			Address:                  "127.0.0.1",
			Port:                     9090,
			ReadTimeoutSeconds:       5,
			ReadHeaderTimeoutSeconds: 6,
			WriteTimeoutSeconds:      7,
			IdleTimeoutSeconds:       8,
		},
	}
	_, srv := createHTTPServerAndRouter(cfg)
	if srv.Addr != "127.0.0.1:9090" {
		t.Fatalf("unexpected addr: %s", srv.Addr)
	}
	if srv.ReadTimeout != 5*time.Second {
		t.Fatalf("unexpected read timeout: %s", srv.ReadTimeout)
	}
	if srv.ReadHeaderTimeout != 6*time.Second {
		t.Fatalf("unexpected read header timeout: %s", srv.ReadHeaderTimeout)
	}
	if srv.WriteTimeout != 7*time.Second {
		t.Fatalf("unexpected write timeout: %s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 8*time.Second {
		t.Fatalf("unexpected idle timeout: %s", srv.IdleTimeout)
	}
}

func TestCreateChiServerOptionsContainsRoutes(t *testing.T) {
	router, _ := createHTTPServerAndRouter(AppConfig{})
	opts := createChiServerOptions(router, nil, AppConfig{}, nil)
	if len(opts.Middlewares["common"]) == 0 {
		t.Fatalf("common middlewares empty")
	}
	keys := []string{
		"POST /items",
		"PUT /items/{id}",
		"GET /items",
		"GET /items/{id}",
		"DELETE /items/{id}",
		"POST /files/presign",
		"GET /files/{fileId}",
	}
	for _, k := range keys {
		if _, ok := opts.Middlewares[k]; !ok {
			t.Fatalf("missing key: %s", k)
		}
	}
}

func TestCreateJWTAuth_EmptySecret(t *testing.T) {
	_, err := createJWTAuth(AppConfig{})
	if err == nil {
		t.Fatalf("expected error for empty secret")
	}
}

func TestCreateApplicationLogger(t *testing.T) {
	logger, err := createApplicationLogger()
	if err != nil {
		t.Fatalf("logger error: %v", err)
	}
	if logger == nil {
		t.Fatalf("logger is nil")
	}
}

func TestCreateDatabaseClientInvalidDSN(t *testing.T) {
	_, err := createDatabaseClient(context.Background(), AppConfig{DB: DatabaseConfig{DataSourceName: "invalid_dsn_string"}})
	if err == nil {
		t.Fatalf("expected error for invalid dsn")
	}
}

func TestCreateServerImplementationForTests_NoS3Configured(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()
	container := &ApplicationContainer{
		router:         chi.NewRouter(),
		logger:         zap.NewNop(),
		databaseClient: dbClient,
		httpServer:     &http.Server{},
		configuration: AppConfig{
			Auth: AuthConfig{AccessTokenTTLSeconds: 3600, RefreshTokenTTLSeconds: 3600, JwtSecret: "x"},
			S3:   S3Config{},
		},
	}
	tokenAuth, err := createJWTAuth(container.configuration)
	require.NoError(t, err)
	server := CreateServerImplementationForTests(container, tokenAuth)

	userID := uuid.New()
	fileID := uuid.New()
	body := map[string]interface{}{"filename": "file.bin", "mime": "application/octet-stream", "checksum": "abc", "fileId": fileID.String()}
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, data)
	server.PresignFile(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	users := repository.NewUserRepository(dbClient)
	_, err = users.Save(t.Context(), model.NewUser(userID, "cfg_user_"+userID.String(), "hash", time.Now().UTC()))
	require.NoError(t, err)
	createBody := map[string]interface{}{"title": "note", "data": map[string]interface{}{"type": "TEXT", "text": "hello"}}
	createBytes, _ := json.Marshal(createBody)
	rec2 := httptest.NewRecorder()
	req2 := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, createBytes)
	server.CreateItem(rec2, req2)
	require.Equal(t, http.StatusCreated, rec2.Code)
}

func TestCreateServerImplementationProviderNil(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()
	container := &ApplicationContainer{
		router:         chi.NewRouter(),
		logger:         zap.NewNop(),
		databaseClient: dbClient,
		httpServer:     &http.Server{},
		configuration: AppConfig{
			Auth:   AuthConfig{AccessTokenTTLSeconds: 3600, RefreshTokenTTLSeconds: 3600, JwtSecret: "x"},
			Crypto: CryptoConfig{},
			S3:     S3Config{Endpoint: "", Bucket: "", AccessKey: "", SecretKey: ""},
		},
	}
	tokenAuth, err := createJWTAuth(container.configuration)
	require.NoError(t, err)
	server := createServerImplementation(container, tokenAuth)
	userID := uuid.New()
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items", userID, nil)
	server.GetItems(rec, req, api.GetItemsParams{})
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestApplicationContainerCloseNilSafe(t *testing.T) {
	var c *ApplicationContainer
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error")
	}
	c = &ApplicationContainer{}
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error for empty container")
	}
}
