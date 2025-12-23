package files

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestWebhookHandlerInvalidSecret(t *testing.T) {
	items := &repository.ItemRepository{}
	s3service := &testutil.S3ServiceStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewWebhookHandler(items, s3service, kek, "expected", "protected")

	request := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader([]byte("{}")))
	request.Header.Set("Authorization", "Bearer wrong")
	response := httptest.NewRecorder()

	handler.Handle(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestWebhookHandlerInvalidJSON(t *testing.T) {
	items := &repository.ItemRepository{}
	s3service := &testutil.S3ServiceStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewWebhookHandler(items, s3service, kek, "", "protected")

	request := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader([]byte("{")))
	response := httptest.NewRecorder()

	handler.Handle(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestWebhookHandlerMissingMetadataTriggersRemoval(t *testing.T) {
	items := &repository.ItemRepository{}
	s3service := &testutil.S3ServiceStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewWebhookHandler(items, s3service, kek, "", "protected")

	event := map[string]interface{}{
		"eventName": "s3:ObjectCreated:Put",
		"Records": []map[string]interface{}{
			{
				"S3": map[string]interface{}{
					"bucket": map[string]interface{}{"name": "bucket"},
					"object": map[string]interface{}{
						"key":         "uploads/x",
						"contentType": "application/octet-stream",
						"size":        10,
						"userMetadata": map[string]string{
							"user-id": "",
							"file-id": "",
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(event)
	request := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader(body))
	response := httptest.NewRecorder()

	handler.Handle(response, request)
	if len(s3service.RemovedRecords) == 0 {
		t.Fatalf("expected removal to be recorded")
	}
}

func TestWebhookHandlerInvalidUUIDsTriggersRemoval(t *testing.T) {
	items := &repository.ItemRepository{}
	s3service := &testutil.S3ServiceStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewWebhookHandler(items, s3service, kek, "", "protected")

	event := map[string]interface{}{
		"eventName": "s3:ObjectCreated:Put",
		"Records": []map[string]interface{}{
			{
				"S3": map[string]interface{}{
					"bucket": map[string]interface{}{"name": "bucket"},
					"object": map[string]interface{}{
						"key":         "uploads/x",
						"contentType": "application/octet-stream",
						"size":        10,
						"userMetadata": map[string]string{
							"user-id":  "x",
							"file-id":  "y",
							"filename": "f.bin",
							"mime":     "application/octet-stream",
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(event)
	request := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader(body))
	response := httptest.NewRecorder()

	handler.Handle(response, request)
	if len(s3service.RemovedRecords) == 0 {
		t.Fatalf("expected removal to be recorded")
	}
}

func TestWebhookHandlerGetObjectErrorTriggersRemoval(t *testing.T) {
	items := &repository.ItemRepository{}
	s3service := &testutil.S3ServiceStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewWebhookHandler(items, s3service, kek, "", "protected")

	userID := uuid.New()
	fileID := uuid.New()
	event := map[string]interface{}{
		"eventName": "s3:ObjectCreated:Put",
		"Records": []map[string]interface{}{
			{
				"S3": map[string]interface{}{
					"bucket": map[string]interface{}{"name": "bucket"},
					"object": map[string]interface{}{
						"key":         "uploads/" + fileID.String(),
						"contentType": "application/octet-stream",
						"size":        10,
						"userMetadata": map[string]string{
							"user-id":  userID.String(),
							"file-id":  fileID.String(),
							"checksum": "abc",
							"mime":     "application/octet-stream",
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(event)
	request := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader(body))
	response := httptest.NewRecorder()

	handler.Handle(response, request)
	if len(s3service.RemovedRecords) == 0 {
		t.Fatalf("expected removal to be recorded")
	}
}

func TestWebhookHandlerPutObjectErrorTriggersRemoval(t *testing.T) {
	items := &repository.ItemRepository{}
	s3service := &testutil.S3ServiceStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewWebhookHandler(items, s3service, kek, "", "protected")

	userID := uuid.New()
	fileID := uuid.New()
	event := map[string]interface{}{
		"eventName": "s3:ObjectCreated:Put",
		"Records": []map[string]interface{}{
			{
				"S3": map[string]interface{}{
					"bucket": map[string]interface{}{"name": "bucket"},
					"object": map[string]interface{}{
						"key":         "uploads/" + fileID.String(),
						"contentType": "application/octet-stream",
						"size":        10,
						"userMetadata": map[string]string{
							"user-id":  userID.String(),
							"file-id":  fileID.String(),
							"checksum": "abc",
							"mime":     "application/octet-stream",
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(event)
	request := httptest.NewRequest(http.MethodPost, "/files/webhook-minio", bytes.NewReader(body))
	response := httptest.NewRecorder()
	s3service.PutError = io.EOF
	handler.Handle(response, request)
	if len(s3service.RemovedRecords) == 0 {
		t.Fatalf("expected removal to be recorded")
	}
}
