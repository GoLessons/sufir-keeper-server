package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestRegisterHandlerIntegration(t *testing.T) {
	databaseClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = databaseClient.Close() }()

	usersRepository := repository.NewUserRepository(databaseClient)
	handler := NewRegisterHandler(usersRepository)

	uniqueLogin := "register_user_" + uuid.New().String()
	requestBody := map[string]string{"login": uniqueLogin, "password": "StrongPassword123!"}
	requestBytes, _ := json.Marshal(requestBody)

	httpRecorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(requestBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.Handle(httpRecorder, httpRequest)

	require.Equal(t, http.StatusCreated, httpRecorder.Code)
	var response map[string]string
	require.NoError(t, json.Unmarshal(httpRecorder.Body.Bytes(), &response))

	record, err := usersRepository.GetByLogin(t.Context(), uniqueLogin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, record.ID)
	require.NotEmpty(t, record.PasswordHash)
	require.WithinDuration(t, time.Now().UTC(), record.CreatedAt, time.Minute)
}
