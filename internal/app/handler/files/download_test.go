package files

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/sio"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

const octetStream = "application/octet-stream"

type DownloadHandlerItemStoreStub struct {
	Error  error
	Record model.ItemRecord
}

func TestDownloadHandlerForbiddenCardType(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	record := model.ItemRecord{
		ID:     itemID,
		UserID: userID,
		Title:  "card.bin",
		Type:   "CARD",
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewDownloadHandler(itemStore, s3service, provider)
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestDownloadHandlerForbiddenCredentialType(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	record := model.ItemRecord{
		ID:     itemID,
		UserID: userID,
		Title:  "cred.bin",
		Type:   "CREDENTIAL",
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewDownloadHandler(itemStore, s3service, provider)
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestDownloadHandlerBinaryInlineSuccess(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	plaintext := []byte("binary inline payload")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encryptedData, err := aead.Encrypt(dek, dataAAD, plaintext)
	if err != nil {
		t.Fatalf("encrypt data error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "inline.bin",
		Type:             "BINARY",
		DataEncrypted:    encryptedData,
		DataNonce:        dataNonce,
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       10,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Meta:             map[string]string{"mime": "application/octet-stream"},
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(kek, 10)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != octetStream {
		t.Fatalf("unexpected content type: %s", ct)
	}
	if disp := rr.Header().Get("Content-Disposition"); !strings.Contains(disp, "inline.bin") {
		t.Fatalf("unexpected content-disposition: %s", disp)
	}
	if !bytes.Equal(rr.Body.Bytes(), plaintext) {
		t.Fatalf("unexpected body")
	}
}

func TestDownloadHandlerFilenameSanitized(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	plaintext := []byte("content")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encryptedData, err := aead.Encrypt(dek, dataAAD, plaintext)
	if err != nil {
		t.Fatalf("encrypt data error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "../dir/evil.bin",
		Type:             "BINARY",
		DataEncrypted:    encryptedData,
		DataNonce:        dataNonce,
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       13,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Meta:             map[string]string{"mime": "application/octet-stream"},
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(kek, 13)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if disp := rr.Header().Get("Content-Disposition"); !strings.Contains(disp, "evil.bin") || strings.Contains(disp, "../") {
		t.Fatalf("unexpected content-disposition: %s", disp)
	}
}

func TestDownloadHandlerContentTypeFallbackWhenMimeMissing(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	plaintext := []byte("content")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encryptedData, err := aead.Encrypt(dek, dataAAD, plaintext)
	if err != nil {
		t.Fatalf("encrypt data error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "file.bin",
		Type:             "BINARY",
		DataEncrypted:    encryptedData,
		DataNonce:        dataNonce,
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       14,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Meta:             map[string]string{},
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(kek, 14)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != octetStream {
		t.Fatalf("unexpected content type: %s", ct)
	}
	if disp := rr.Header().Get("Content-Disposition"); !strings.Contains(disp, "file.bin") {
		t.Fatalf("unexpected content-disposition: %s", disp)
	}
}

func (stub *DownloadHandlerItemStoreStub) Create(_ context.Context, _ model.ItemRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, nil
}

func (stub *DownloadHandlerItemStoreStub) Get(_ context.Context, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	if stub.Error != nil {
		return model.ItemRecord{}, stub.Error
	}
	return stub.Record, nil
}

func (stub *DownloadHandlerItemStoreStub) List(_ context.Context, _ uuid.UUID, _ *string, _ *string, _ int, _ int) (repository.ListResult, error) {
	return repository.ListResult{Total: 0, Items: nil}, nil
}

func (stub *DownloadHandlerItemStoreStub) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ repository.ItemUpdateRecord) (model.ItemRecord, error) {
	return stub.Record, nil
}

func (stub *DownloadHandlerItemStoreStub) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (stub *DownloadHandlerItemStoreStub) BeginTx(_ context.Context) (*sql.Tx, error) {
	return nil, nil // using nil tx for stubbed paths
}

func (stub *DownloadHandlerItemStoreStub) GetWithTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	return stub.Record, nil
}

func (stub *DownloadHandlerItemStoreStub) DeleteWithTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type DownloadHandlerKEKProviderErrorStub struct{}

func (p *DownloadHandlerKEKProviderErrorStub) GetCurrent(_ context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("error")
}

func (p *DownloadHandlerKEKProviderErrorStub) GetByVersion(_ context.Context, _ int) ([]byte, error) {
	return nil, errors.New("error")
}

func (p *DownloadHandlerKEKProviderErrorStub) Rotate(_ context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("error")
}

func TestDownloadHandlerUnauthorized(t *testing.T) {
	itemStore := &DownloadHandlerItemStoreStub{}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/files/"+id.String(), nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, id)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestDownloadHandlerNotFoundRecord(t *testing.T) {
	itemStore := &DownloadHandlerItemStoreStub{Error: errors.New("db")}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	id := uuid.New()
	userID := uuid.New()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+id.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, id)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDownloadHandlerForbiddenType(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	record := model.ItemRecord{
		ID:        itemID,
		UserID:    userID,
		Title:     "text-item",
		Type:      "TEXT",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestDownloadHandlerKEKError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "binary-kek-error",
		Type:             "BINARY",
		DataKeyEncrypted: []byte("x"),
		DataKeyNonce:     []byte("y"),
		KEKVersion:       5,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := &DownloadHandlerKEKProviderErrorStub{}
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDownloadHandlerDecryptKeyError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek1 := make([]byte, 32)
	_, _ = rand.Read(kek1)
	kek2 := make([]byte, 32)
	_, _ = rand.Read(kek2)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	nonce, encryptedKey, err := aead.Encrypt(kek1, keyAAD, []byte("short-dek"))
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "binary-decrypt-key-error",
		Type:             "BINARY",
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     nonce,
		KEKVersion:       7,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(kek2, 7)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDownloadHandlerFileObjectNotFound(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	nonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "binary-file-object-not-found",
		Type:             "BINARY",
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     nonce,
		KEKVersion:       9,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		File: &model.ItemFile{
			S3Bucket: "bucket",
			S3Key:    "key",
			Size:     100,
			SHA256:   "",
		},
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{}
	provider := keyencrypt.NewStaticProvider(kek, 9)
	handler := NewDownloadHandler(itemStore, s3service, provider)

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDownloadHandlerSuccessStreamDecryption(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	plaintext := bytes.Repeat([]byte("data-"), 1000)
	src := bytes.NewReader(plaintext)
	encReader, err := sio.EncryptReader(src, sio.Config{
		Key:          dek,
		MinVersion:   sio.Version20,
		CipherSuites: []byte{sio.AES_256_GCM},
	})
	if err != nil {
		t.Fatalf("encrypt reader error: %v", err)
	}
	encryptedBytes, err := io.ReadAll(encReader)
	if err != nil {
		t.Fatalf("read encrypted error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "stream.bin",
		Type:             "BINARY",
		DataEncrypted:    nil,
		DataNonce:        nil,
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       20,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Meta:             map[string]string{"mime": "application/octet-stream"},
		File: &model.ItemFile{
			S3Bucket: "protected",
			S3Key:    "key",
			Size:     int64(len(encryptedBytes)),
			SHA256:   "",
		},
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{GetObjectData: encryptedBytes, StatSize: int64(len(encryptedBytes))}
	provider := keyencrypt.NewStaticProvider(kek, 20)
	handler := NewDownloadHandler(itemStore, s3service, provider)
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !bytes.Equal(rr.Body.Bytes(), plaintext) {
		t.Fatalf("unexpected body")
	}
}

func TestDownloadHandlerMetaHeaders(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("key encrypt error: %v", err)
	}
	data := []byte("hello")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encData, err := aead.Encrypt(dek, dataAAD, data)
	if err != nil {
		t.Fatalf("data encrypt error: %v", err)
	}
	itemStore := &DownloadHandlerItemStoreStub{
		Record: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "",
			Type:             "BINARY",
			DataEncrypted:    encData,
			DataNonce:        dataNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       1,
			Meta:             map[string]string{"filename": " a/b/c \n ", "mime": "text/plain"},
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	handler := NewDownloadHandler(itemStore, &testutil.S3ServiceStub{}, keyencrypt.NewStaticProvider(kek, 1))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("unexpected content type: %s", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "filename=\"c\"") {
		t.Fatalf("unexpected content disposition: %s", cd)
	}
}

func TestDownloadHandlerForbiddenForNonBinaryFile(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("key encrypt error: %v", err)
	}
	itemStore := &DownloadHandlerItemStoreStub{
		Record: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "x",
			Type:             "TEXT",
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       1,
			File:             &model.ItemFile{S3Bucket: "bucket", S3Key: "key", Size: 1},
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	handler := NewDownloadHandler(itemStore, &testutil.S3ServiceStub{}, keyencrypt.NewStaticProvider(kek, 1))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestDownloadHandlerS3NotFound(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("key encrypt error: %v", err)
	}
	itemStore := &DownloadHandlerItemStoreStub{
		Record: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "bin",
			Type:             "BINARY",
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       1,
			File:             &model.ItemFile{S3Bucket: "bucket", S3Key: "key", Size: 5},
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	handler := NewDownloadHandler(itemStore, &testutil.S3ServiceStub{}, keyencrypt.NewStaticProvider(kek, 1))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDownloadHandlerDecryptDataError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("key encrypt error: %v", err)
	}
	data := []byte("hello")
	wrongNonce := make([]byte, len(keyNonce))
	copy(wrongNonce, keyNonce)
	itemStore := &DownloadHandlerItemStoreStub{
		Record: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "bin",
			Type:             "BINARY",
			DataEncrypted:    data,
			DataNonce:        wrongNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       1,
			File:             nil,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	handler := NewDownloadHandler(itemStore, &testutil.S3ServiceStub{}, keyencrypt.NewStaticProvider(kek, 1))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDownloadHandlerTitleSanitize(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, dek)
	if err != nil {
		t.Fatalf("key encrypt error: %v", err)
	}
	data := []byte("hi")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encData, err := aead.Encrypt(dek, dataAAD, data)
	if err != nil {
		t.Fatalf("data encrypt error: %v", err)
	}
	itemStore := &DownloadHandlerItemStoreStub{
		Record: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            " ../a/b/c\r\n ",
			Type:             "BINARY",
			DataEncrypted:    encData,
			DataNonce:        dataNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       1,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	handler := NewDownloadHandler(itemStore, &testutil.S3ServiceStub{}, keyencrypt.NewStaticProvider(kek, 1))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "filename=\"c\"") {
		t.Fatalf("unexpected content disposition: %s", cd)
	}
}

func TestDownloadHandlerDecryptInitError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	_, _ = rand.Read(kek)
	dekShort := bytes.Repeat([]byte{1}, 16)
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encryptedKey, err := aead.Encrypt(kek, keyAAD, dekShort)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	encryptedBytes := bytes.Repeat([]byte{2}, 1024)
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "stream-error.bin",
		Type:             "BINARY",
		DataKeyEncrypted: encryptedKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       21,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Meta:             map[string]string{},
		File: &model.ItemFile{
			S3Bucket: "protected",
			S3Key:    "key",
			Size:     int64(len(encryptedBytes)),
			SHA256:   "",
		},
	}
	itemStore := &DownloadHandlerItemStoreStub{Record: record}
	s3service := &testutil.S3ServiceStub{GetObjectData: encryptedBytes, StatSize: int64(len(encryptedBytes))}
	provider := keyencrypt.NewStaticProvider(kek, 21)
	handler := NewDownloadHandler(itemStore, s3service, provider)
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
