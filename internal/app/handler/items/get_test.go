package items

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

type GetItemsStoreStub struct {
	Err    error
	Record model.ItemRecord
}

func (s *GetItemsStoreStub) Create(ctx context.Context, rec model.ItemRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *GetItemsStoreStub) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	if s.Err != nil {
		return model.ItemRecord{}, s.Err
	}
	return s.Record, nil
}

func (s *GetItemsStoreStub) List(ctx context.Context, userID uuid.UUID, filterType *string, search *string, limit int, offset int) (repository.ListResult, error) {
	return repository.ListResult{}, errors.New("not implemented")
}

func (s *GetItemsStoreStub) Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, upd repository.ItemUpdateRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *GetItemsStoreStub) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (s *GetItemsStoreStub) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (s *GetItemsStoreStub) GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *GetItemsStoreStub) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

type ErrorVersionKeyEncryptProvider struct{}

func (e *ErrorVersionKeyEncryptProvider) GetCurrent(ctx context.Context) ([]byte, int, error) {
	return make([]byte, 32), 1, nil
}

func (e *ErrorVersionKeyEncryptProvider) GetByVersion(ctx context.Context, version int) ([]byte, error) {
	return nil, errors.New("kek")
}

func (e *ErrorVersionKeyEncryptProvider) Rotate(ctx context.Context) ([]byte, int, error) {
	return make([]byte, 32), 2, nil
}

func TestGetHandlerUnauthorized(t *testing.T) {
	itemsRepo := &GetItemsStoreStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewGetHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	req := httptest.NewRequest(http.MethodGet, "/items/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestGetHandlerNotFound(t *testing.T) {
	itemsRepo := &GetItemsStoreStub{Err: errors.New("not found")}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewGetHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	itemID := uuid.New()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetHandlerKekError(t *testing.T) {
	itemsRepo := &GetItemsStoreStub{Record: model.ItemRecord{ID: uuid.New(), UserID: uuid.New(), KEKVersion: 10, Type: "TEXT"}}
	handler := NewGetHandler(itemsRepo, &ErrorVersionKeyEncryptProvider{}, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	itemID := itemsRepo.Record.ID
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestGetHandlerDecryptKeyError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kekBytes := make([]byte, 32)
	for i := range kekBytes {
		kekBytes[i] = byte(1 + i%251)
	}
	itemKey := []byte("01234567890123456789012345678901")
	keyAADWrong := []byte(uuid.New().String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kekBytes, keyAADWrong, itemKey)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "t",
		Type:             "TEXT",
		DataKeyEncrypted: encKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       3,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	itemsRepo := &GetItemsStoreStub{Record: record}
	handler := NewGetHandler(itemsRepo, keyencrypt.NewStaticProvider(kekBytes, 3), jwtauth.New("HS256", []byte("secret"), nil))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestGetHandlerDecryptDataError(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kekBytes := make([]byte, 32)
	for i := range kekBytes {
		kekBytes[i] = byte(3 + i%251)
	}
	itemKey := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kekBytes, keyAAD, itemKey)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	dataAADWrong := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAADWrong, []byte(`{"type":"TEXT","value":"hello"}`))
	if err != nil {
		t.Fatalf("encrypt data error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "t",
		Type:             "TEXT",
		DataEncrypted:    encData,
		DataNonce:        dataNonce,
		DataKeyEncrypted: encKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       4,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	itemsRepo := &GetItemsStoreStub{Record: record}
	handler := NewGetHandler(itemsRepo, keyencrypt.NewStaticProvider(kekBytes, 4), jwtauth.New("HS256", []byte("secret"), nil))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestGetHandlerSuccessText(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kekBytes := make([]byte, 32)
	for i := range kekBytes {
		kekBytes[i] = byte(5 + i%251)
	}
	itemKey := []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kekBytes, keyAAD, itemKey)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "TEXT")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAAD, []byte(`{"type":"TEXT","value":"hello"}`))
	if err != nil {
		t.Fatalf("encrypt data error: %v", err)
	}
	record := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "title",
		Type:             "TEXT",
		DataEncrypted:    encData,
		DataNonce:        dataNonce,
		DataKeyEncrypted: encKey,
		DataKeyNonce:     keyNonce,
		KEKVersion:       6,
		Meta:             map[string]string{"k": "v"},
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	itemsRepo := &GetItemsStoreStub{Record: record}
	handler := NewGetHandler(itemsRepo, keyencrypt.NewStaticProvider(kekBytes, 6), jwtauth.New("HS256", []byte("secret"), nil))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("unexpected empty body")
	}
}
