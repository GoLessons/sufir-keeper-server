package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/auth"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestLoginHandlerIntegration(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()

	usersRepository := repository.NewUserRepository(databaseClient)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)
	handler := NewLoginHandler(usersRepository, tokenAuth, 3600, 2592000)

	login := "login_integration_user_" + uuid.New().String()
	password := "StrongPassword123!"
	hashed, err := auth.HashPassword(password)
	require.NoError(t, err)
	_, err = usersRepository.Save(t.Context(), model.NewUser(uuid.New(), login, hashed, time.Now().UTC()))
	require.NoError(t, err)

	body := map[string]string{"login": login, "password": password}
	bodyBytes, _ := json.Marshal(body)
	httpRecorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(bodyBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.Handle(httpRecorder, httpRequest)

	require.Equal(t, http.StatusOK, httpRecorder.Code)
	var resp apitypes.AuthResponse
	require.NoError(t, json.Unmarshal(httpRecorder.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)
	require.NotEmpty(t, resp.TokenType)
	require.NotEmpty(t, resp.ExpiresIn)
}

func TestLoginHandlerInvalidJSON(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()
	usersRepository := repository.NewUserRepository(databaseClient)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)
	handler := NewLoginHandler(usersRepository, tokenAuth, 3600, 2592000)
	httpRecorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader([]byte(`{`)))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.Handle(httpRecorder, httpRequest)
	require.Equal(t, http.StatusBadRequest, httpRecorder.Code)
}

func TestLoginHandlerUnknownLogin(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()
	usersRepository := repository.NewUserRepository(databaseClient)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)
	handler := NewLoginHandler(usersRepository, tokenAuth, 3600, 2592000)
	requestBody := map[string]string{"login": "unknown_login_" + uuid.New().String(), "password": "StrongPassword123!"}
	requestBytes, _ := json.Marshal(requestBody)
	httpRecorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(requestBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.Handle(httpRecorder, httpRequest)
	require.Equal(t, http.StatusUnauthorized, httpRecorder.Code)
}

func TestLoginHandlerWrongPassword(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()
	usersRepository := repository.NewUserRepository(databaseClient)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)
	handler := NewLoginHandler(usersRepository, tokenAuth, 3600, 2592000)
	login := "wrong_password_user_" + uuid.New().String()
	_, err := usersRepository.Save(t.Context(), model.NewUser(uuid.New(), login, "hashed", time.Now().UTC()))
	require.NoError(t, err)
	requestBody := map[string]string{"login": login, "password": "invalid"}
	requestBytes, _ := json.Marshal(requestBody)
	httpRecorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(requestBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.Handle(httpRecorder, httpRequest)
	require.Equal(t, http.StatusUnauthorized, httpRecorder.Code)
}
