package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestLogoutHandlerIntegration(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()

	usersRepository := repository.NewUserRepository(databaseClient)
	handler := NewLogoutHandler(usersRepository, nil)

	login := "logout_integration_user_" + uuid.New().String()
	_, err := usersRepository.Save(t.Context(), model.NewUser(uuid.New(), login, "dummy", time.Now().UTC()))
	require.NoError(t, err)
	userRecord, err := usersRepository.GetByLogin(t.Context(), login)
	require.NoError(t, err)
	oldVersion, err := usersRepository.EnsureRefreshVersion(t.Context(), userRecord.ID)
	require.NoError(t, err)

	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/auth", userRecord.ID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)

	newVersion, err := usersRepository.GetRefreshVersion(t.Context(), userRecord.ID)
	require.NoError(t, err)
	require.Greater(t, newVersion, oldVersion)
}

func TestLogoutHandlerUnauthorized(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()
	usersRepository := repository.NewUserRepository(databaseClient)
	handler := NewLogoutHandler(usersRepository, nil)
	req := httptest.NewRequest(http.MethodDelete, "/auth", nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
