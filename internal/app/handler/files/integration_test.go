package files

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/auth"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

type FilesIntegrationFakeS3 = testutil.S3ServiceStub

func createDatabaseClientForFilesIntegration(t *testing.T) *db.Client {
	return testutil.CreateDatabaseClientForIntegrationTests(t)
}

func createAuthorizedRequest(method string, path string, userID uuid.UUID, body []byte) *http.Request {
	return testutil.AuthorizedJSONRequest(method, path, userID, body)
}

func TestPresignHandlerIntegration(t *testing.T) {
	dbClient := createDatabaseClientForFilesIntegration(t)
	defer func() { _ = dbClient.Close() }()

	fakeS3 := &FilesIntegrationFakeS3{}
	handler := NewPresignHandler(fakeS3)

	userIdentifier := uuid.New()
	fileIdentifier := uuid.New()
	requestBody := map[string]interface{}{
		"fileId":   fileIdentifier.String(),
		"filename": "integration.txt",
		"mime":     "text/plain",
		"checksum": "abc",
	}
	bodyBytes, _ := json.Marshal(requestBody)
	rec := httptest.NewRecorder()
	req := createAuthorizedRequest(http.MethodPost, "/files/presign", userIdentifier, bodyBytes)

	handler.Handle(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		FormFields map[string]string `json:"form_fields"`
		UploadURL  string            `json:"upload_url"`
		Key        string            `json:"key"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "/api/v1/files", resp.UploadURL)
	require.Equal(t, fileIdentifier.String(), resp.Key)
	require.NotEmpty(t, resp.FormFields["key"])
	require.Len(t, fakeS3.PresignedCalls, 1)
	require.Equal(t, fileIdentifier.String(), fakeS3.PresignedCalls[0].Key)
}

func TestDownloadHandlerBinaryInlineIntegration(t *testing.T) {
	dbClient := createDatabaseClientForFilesIntegration(t)
	defer func() { _ = dbClient.Close() }()

	itemRepository := repository.NewItemRepository(dbClient)
	userRepository := repository.NewUserRepository(dbClient)
	keyProvider := keyencrypt.NewStaticProvider(make([]byte, 32), 3)
	fakeS3 := &FilesIntegrationFakeS3{}
	handler := NewDownloadHandler(itemRepository, fakeS3, keyProvider)

	userIdentifier := uuid.New()
	passwordHash, err := auth.HashPassword("StrongPassword123!")
	require.NoError(t, err)
	uniqueLogin := "files_integration_user_" + uuid.New().String()
	userModel := model.NewUser(userIdentifier, uniqueLogin, passwordHash, time.Now().UTC())
	_, err = userRepository.Save(t.Context(), userModel)
	require.NoError(t, err)
	itemIdentifier := uuid.New()
	plaintext := []byte("integration-binary-content")
	dataEncryptionKey := make([]byte, 32)
	_, err = rand.Read(dataEncryptionKey)
	require.NoError(t, err)
	dataAAD := []byte(userIdentifier.String() + "|" + itemIdentifier.String() + "|" + "BINARY")
	dataNonce, encryptedData, err := aead.Encrypt(dataEncryptionKey, dataAAD, plaintext)
	require.NoError(t, err)
	kek, kekVersion, err := keyProvider.GetCurrent(t.Context())
	require.NoError(t, err)
	keyAAD := []byte(userIdentifier.String() + "|" + itemIdentifier.String())
	keyNonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, dataEncryptionKey)
	require.NoError(t, err)

	record := model.ItemRecord{
		ID:               itemIdentifier,
		UserID:           userIdentifier,
		Title:            "binary-inline",
		Type:             "BINARY",
		DataEncrypted:    encryptedData,
		DataNonce:        dataNonce,
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       kekVersion,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	_, err = itemRepository.Create(t.Context(), record)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := createAuthorizedRequest(http.MethodGet, "/files/"+itemIdentifier.String(), userIdentifier, nil)
	handler.Handle(rec, req, itemIdentifier)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	require.Equal(t, plaintext, rec.Body.Bytes())
}
