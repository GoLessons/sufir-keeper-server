package files

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestWebhookHandlerIntegration(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()

	items := repository.NewItemRepository(dbClient)
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 4)
	fakeS3 := &testutil.S3ServiceStub{PresignedCalls: nil}
	protectedBucket := "keeper-protected"
	secret := "dev-webhook-secret"
	handler := NewWebhookHandler(items, fakeS3, kek, secret, protectedBucket)

	userID := uuid.New()
	fileID := uuid.New()
	key := "uploads/" + fileID.String()

	event := map[string]interface{}{
		"eventName": "s3:ObjectCreated:Put",
		"Records": []map[string]interface{}{
			{
				"S3": map[string]interface{}{
					"bucket": map[string]interface{}{"name": "keeper"},
					"object": map[string]interface{}{
						"key":         key,
						"contentType": "application/octet-stream",
						"size":        20,
						"userMetadata": map[string]string{
							"user-id":  userID.String(),
							"file-id":  fileID.String(),
							"filename": "file.bin",
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

	require.Len(t, fakeS3.RemovedRecords, 1)
	require.Equal(t, "", fakeS3.RemovedRecords[0].Bucket)
	require.Equal(t, key, fakeS3.RemovedRecords[0].Key)
}
