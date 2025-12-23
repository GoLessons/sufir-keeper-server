package files

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/sio"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestWebhookHandlerSuccessStreamingEncryption(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()

	items := repository.NewItemRepository(dbClient)
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 5)
	protectedBucket := "keeper-protected"
	secret := "dev-webhook-secret"

	plaintext := bytes.Repeat([]byte("payload-"), 2000)
	fakeS3 := &testutil.S3ServiceStub{
		GetObjectData: plaintext,
		StatSize:      int64(len(plaintext)),
	}
	handler := NewWebhookHandler(items, fakeS3, kek, secret, protectedBucket)

	userID := uuid.New()
	fileID := uuid.New()
	key := "uploads/" + fileID.String()

	userRepo := repository.NewUserRepository(dbClient)
	_, err := userRepo.Save(context.Background(), model.User{
		ID:           userID,
		Login:        "user_" + userID.String(),
		PasswordHash: "hash",
		CreatedAt:    time.Now().UTC(),
	})
	require.NoError(t, err)

	event := map[string]interface{}{
		"eventName": "s3:ObjectCreated:Put",
		"Records": []map[string]interface{}{
			{
				"S3": map[string]interface{}{
					"bucket": map[string]interface{}{"name": "keeper"},
					"object": map[string]interface{}{
						"key":         key,
						"contentType": "application/octet-stream",
						"size":        len(plaintext),
						"userMetadata": map[string]string{
							"user-id":  userID.String(),
							"file-id":  fileID.String(),
							"filename": "f.bin",
							"mime":     "application/octet-stream",
							"checksum": "abc",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+secret)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var putProtected *testutil.S3ServiceStubPutCall
	for i := range fakeS3.PutCalls {
		call := fakeS3.PutCalls[i]
		if call.Bucket == protectedBucket && call.Key == key {
			putProtected = &call
			break
		}
	}
	require.NotNil(t, putProtected)

	encryptedSize, err := sio.EncryptedSize(uint64(len(plaintext)))
	require.NoError(t, err)
	require.Equal(t, int64(encryptedSize), putProtected.Size)

	var removedSource *testutil.S3ServiceStubRemovedRecord
	for i := range fakeS3.RemovedRecords {
		rec := fakeS3.RemovedRecords[i]
		if rec.Bucket == "" && rec.Key == key {
			removedSource = &rec
			break
		}
	}
	require.NotNil(t, removedSource)

	rec, err := items.Get(req.Context(), userID, fileID)
	require.NoError(t, err)
	require.Equal(t, "BINARY", rec.Type)
	require.NotNil(t, rec.File)
	require.Equal(t, protectedBucket, rec.File.S3Bucket)
	require.Equal(t, key, rec.File.S3Key)
}
