package items

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

type DeleteItemsStoreStub struct {
	repository.ItemStore
	GetWithTxError  error
	DeleteError     error
	GetWithTxRecord model.ItemRecord
}

func (s *DeleteItemsStoreStub) BeginTx(ctx context.Context) (*sql.Tx, error) {
	client, err := db.NewClient(ctx, testutil.DefaultIntegrationPostgresDataSourceName, db.Options{})
	if err != nil {
		return nil, err
	}
	return client.SQL.BeginTx(ctx, &sql.TxOptions{})
}

func (s *DeleteItemsStoreStub) GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	if s.GetWithTxError != nil {
		return model.ItemRecord{}, s.GetWithTxError
	}
	return s.GetWithTxRecord, nil
}

func (s *DeleteItemsStoreStub) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error {
	return s.DeleteError
}

type ErrorS3ServiceStub struct{}

func (e *ErrorS3ServiceStub) EnsureBucket(_ context.Context, _ string) error { return nil }
func (e *ErrorS3ServiceStub) GetObject(_ context.Context, _ string, _ string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (e *ErrorS3ServiceStub) StatObject(_ context.Context, _ string, _ string) (minio.ObjectInfo, error) {
	return minio.ObjectInfo{}, nil
}

func (e *ErrorS3ServiceStub) RemoveObject(_ context.Context, _ string, _ string) error {
	return errors.New("remove error")
}

func (e *ErrorS3ServiceStub) PutObject(_ context.Context, _ string, _ string, _ io.Reader, _ int64, _ string, _ map[string]string) (minio.UploadInfo, error) {
	return minio.UploadInfo{}, nil
}
func (e *ErrorS3ServiceStub) SetBucketWebhookCreatedEvents(_ context.Context) error { return nil }
func (e *ErrorS3ServiceStub) PresignPost(_ context.Context, _ string, _ string, _ int64, _ map[string]string, _ time.Duration) (string, map[string]string, error) {
	return "", nil, nil
}

func TestDeleteHandlerUnauthorized(t *testing.T) {
	itemsRepo := &DeleteItemsStoreStub{}
	handler := NewDeleteHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil), nil)
	req := httptest.NewRequest(http.MethodDelete, "/items/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestDeleteHandlerNotFound(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemsRepo := &DeleteItemsStoreStub{GetWithTxError: errors.New("not found")}
	handler := NewDeleteHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil), nil)
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteHandlerStorageError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemsRepo := &DeleteItemsStoreStub{
		GetWithTxRecord: model.ItemRecord{ID: itemID, UserID: userID, File: &model.ItemFile{S3Bucket: "bucket", S3Key: "key", Size: 1}},
	}
	s3client := &s3ServiceErrorAdapter{ErrorS3ServiceStub{}}
	handler := NewDeleteHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil), s3client)
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDeleteHandlerSuccess(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemsRepo := &DeleteItemsStoreStub{
		GetWithTxRecord: model.ItemRecord{ID: itemID, UserID: userID, File: nil},
	}
	s3client := &s3ServiceNoopAdapter{}
	handler := NewDeleteHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil), s3client)
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestDeleteHandlerNotFoundOnDelete(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemsRepo := &DeleteItemsStoreStub{
		GetWithTxRecord: model.ItemRecord{ID: itemID, UserID: userID, File: nil},
		DeleteError:     sql.ErrNoRows,
	}
	s3client := &s3ServiceNoopAdapter{}
	handler := NewDeleteHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil), s3client)
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

type s3ServiceErrorAdapter struct{ ErrorS3ServiceStub }

func (a *s3ServiceErrorAdapter) EnsureBucket(ctx context.Context, bucketName string) error {
	return nil
}

func (a *s3ServiceErrorAdapter) GetObject(ctx context.Context, bucketName string, key string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (a *s3ServiceErrorAdapter) StatObject(ctx context.Context, bucketName string, key string) (minio.ObjectInfo, error) {
	return minio.ObjectInfo{}, nil
}

func (a *s3ServiceErrorAdapter) RemoveObject(ctx context.Context, bucketName string, key string) error {
	return errors.New("remove error")
}

func (a *s3ServiceErrorAdapter) PutObject(ctx context.Context, bucketName string, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (minio.UploadInfo, error) {
	return minio.UploadInfo{}, nil
}
func (a *s3ServiceErrorAdapter) SetBucketWebhookCreatedEvents(ctx context.Context) error { return nil }
func (a *s3ServiceErrorAdapter) PresignPost(ctx context.Context, key string, contentType string, size int64, metadata map[string]string, expires time.Duration) (string, map[string]string, error) {
	return "", nil, nil
}

type s3ServiceNoopAdapter struct{}

func (a *s3ServiceNoopAdapter) EnsureBucket(ctx context.Context, bucketName string) error { return nil }
func (a *s3ServiceNoopAdapter) GetObject(ctx context.Context, bucketName string, key string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (a *s3ServiceNoopAdapter) StatObject(ctx context.Context, bucketName string, key string) (minio.ObjectInfo, error) {
	return minio.ObjectInfo{}, nil
}

func (a *s3ServiceNoopAdapter) RemoveObject(ctx context.Context, bucketName string, key string) error {
	return nil
}

func (a *s3ServiceNoopAdapter) PutObject(ctx context.Context, bucketName string, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (minio.UploadInfo, error) {
	return minio.UploadInfo{}, nil
}
func (a *s3ServiceNoopAdapter) SetBucketWebhookCreatedEvents(ctx context.Context) error { return nil }
func (a *s3ServiceNoopAdapter) PresignPost(ctx context.Context, key string, contentType string, size int64, metadata map[string]string, expires time.Duration) (string, map[string]string, error) {
	return "", nil, nil
}
