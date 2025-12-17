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
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestRefreshHandlerIntegration(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()

	usersRepository := repository.NewUserRepository(databaseClient)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)
	handler := NewRefreshHandler(usersRepository, tokenAuth, 3600, 2592000)

	login := "refresh_integration_user_" + uuid.New().String()
	_, err := usersRepository.Save(t.Context(), model.NewUser(uuid.New(), login, "dummy", time.Now().UTC()))
	require.NoError(t, err)
	userRecord, err := usersRepository.GetByLogin(t.Context(), login)
	require.NoError(t, err)
	currentVersion, err := usersRepository.EnsureRefreshVersion(t.Context(), userRecord.ID)
	require.NoError(t, err)

	expiresRefresh := time.Now().Add(10 * time.Minute).Unix()
	_, refreshToken, err := tokenAuth.Encode(map[string]interface{}{"sub": userRecord.ID.String(), "exp": expiresRefresh, "typ": "refresh", "ver": currentVersion})
	require.NoError(t, err)

	body := map[string]string{"refresh_token": refreshToken}
	bodyBytes, _ := json.Marshal(body)
	httpRecorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPatch, "/auth", bytes.NewReader(bodyBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.Handle(httpRecorder, httpRequest)

	require.Equal(t, http.StatusOK, httpRecorder.Code)
	var resp apitypes.AuthResponse
	require.NoError(t, json.Unmarshal(httpRecorder.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)
}
