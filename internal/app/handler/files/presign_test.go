package files

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/s3"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestPresignHandlerUnauthorized(t *testing.T) {
	s3service := &testutil.S3ServiceStub{}
	handler := NewPresignHandler(s3service)

	req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

type FailS3ServiceStub struct{ testutil.S3ServiceStub }

func (f *FailS3ServiceStub) PresignPost(_ context.Context, _ string, _ string, _ int64, _ map[string]string, _ time.Duration) (string, map[string]string, error) {
	return "", nil, errors.New("presign error")
}

func TestPresignHandlerS3Error(t *testing.T) {
	s3service := &FailS3ServiceStub{}
	var _ s3.Service = s3service
	handler := NewPresignHandler(s3service)
	userID := uuid.New()
	fileID := uuid.New()
	body := map[string]interface{}{
		"filename": "file.bin",
		"mime":     "application/octet-stream",
		"checksum": "abc",
		"fileId":   fileID.String(),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestPresignHandlerInvalidFileIDFormat(t *testing.T) {
	s3service := &testutil.S3ServiceStub{}
	handler := NewPresignHandler(s3service)

	userID := uuid.New()
	body := []byte(`{"filename":"f.bin","mime":"application/octet-stream","checksum":"abc","fileId":"not-uuid"}`)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, body)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPresignHandlerOptionalMetadataOmitted(t *testing.T) {
	s3service := &testutil.S3ServiceStub{}
	handler := NewPresignHandler(s3service)

	userID := uuid.New()
	fileID := uuid.New()
	body := map[string]interface{}{
		"filename": "   ",
		"mime":     "   ",
		"checksum": "",
		"fileId":   fileID.String(),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if len(s3service.PresignedCalls) != 1 {
		t.Fatalf("expected single presign call, got %d", len(s3service.PresignedCalls))
	}
	call := s3service.PresignedCalls[0]
	if _, ok := call.Metadata["filename"]; ok {
		t.Fatalf("filename should be omitted")
	}
	if _, ok := call.Metadata["mime"]; ok {
		t.Fatalf("mime should be omitted")
	}
	if _, ok := call.Metadata["checksum"]; ok {
		t.Fatalf("checksum should be omitted")
	}
	if call.Metadata["user-id"] != userID.String() || call.Metadata["file-id"] != fileID.String() {
		t.Fatalf("required metadata missing")
	}
}

func TestPresignHandlerInvalidJSON(t *testing.T) {
	s3service := &testutil.S3ServiceStub{}
	handler := NewPresignHandler(s3service)

	userID := uuid.New()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, []byte(`{`))
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPresignHandlerMissingFileID(t *testing.T) {
	s3service := &testutil.S3ServiceStub{}
	handler := NewPresignHandler(s3service)

	userID := uuid.New()
	body := map[string]interface{}{
		"filename": "file.bin",
		"mime":     "application/octet-stream",
		"checksum": "abc",
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPresignHandlerSuccess(t *testing.T) {
	s3service := &testutil.S3ServiceStub{}
	handler := NewPresignHandler(s3service)

	userID := uuid.New()
	fileID := uuid.New()
	body := map[string]interface{}{
		"filename": "file.bin",
		"mime":     "application/octet-stream",
		"checksum": "abc",
		"fileId":   fileID.String(),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if len(s3service.PresignedCalls) != 1 {
		t.Fatalf("expected single presign call, got %d", len(s3service.PresignedCalls))
	}
	call := s3service.PresignedCalls[0]
	if call.Key != fileID.String() {
		t.Fatalf("unexpected key: %s", call.Key)
	}
	if call.ContentType != "application/octet-stream" {
		t.Fatalf("unexpected content type: %s", call.ContentType)
	}
	if call.Metadata["user-id"] != userID.String() {
		t.Fatalf("missing user-id metadata")
	}
	if call.Metadata["file-id"] != fileID.String() {
		t.Fatalf("missing file-id metadata")
	}
	if call.Metadata["filename"] != "file.bin" {
		t.Fatalf("missing filename metadata")
	}
	if call.Metadata["mime"] != "application/octet-stream" {
		t.Fatalf("missing mime metadata")
	}
	if call.Metadata["checksum"] != "abc" {
		t.Fatalf("missing checksum metadata")
	}
	var resp struct {
		FormFields map[string]string `json:"form_fields"`
		UploadURL  string            `json:"upload_url"`
		Key        string            `json:"key"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.UploadURL != "/api/v1/files" {
		t.Fatalf("unexpected upload url: %s", resp.UploadURL)
	}
	if resp.Key != fileID.String() {
		t.Fatalf("unexpected response key: %s", resp.Key)
	}
	if resp.FormFields["key"] != fileID.String() {
		t.Fatalf("missing form key field")
	}
	if resp.FormFields["success_action_status"] != "204" {
		t.Fatalf("unexpected success action status: %s", resp.FormFields["success_action_status"])
	}
}
