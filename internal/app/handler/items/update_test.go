package items

import (
	"context"
	"database/sql"
	"encoding/json"
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

type UpdateItemsStoreStub struct {
	UpdateError     error
	GetWithTxError  error
	Record          model.ItemRecord
	GetWithTxRecord model.ItemRecord
}

func (s *UpdateItemsStoreStub) Create(ctx context.Context, rec model.ItemRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *UpdateItemsStoreStub) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	return s.Record, nil
}

func (s *UpdateItemsStoreStub) List(ctx context.Context, userID uuid.UUID, filterType *string, search *string, limit int, offset int) (repository.ListResult, error) {
	return repository.ListResult{}, errors.New("not implemented")
}

func (s *UpdateItemsStoreStub) Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, upd repository.ItemUpdateRecord) (model.ItemRecord, error) {
	if s.UpdateError != nil {
		return model.ItemRecord{}, s.UpdateError
	}
	return s.Record, nil
}

func (s *UpdateItemsStoreStub) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (s *UpdateItemsStoreStub) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (s *UpdateItemsStoreStub) GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	if s.GetWithTxError != nil {
		return model.ItemRecord{}, s.GetWithTxError
	}
	if s.GetWithTxRecord.ID != uuid.Nil {
		return s.GetWithTxRecord, nil
	}
	return s.Record, nil
}

func (s *UpdateItemsStoreStub) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

type ErrorCurrentKeyEncryptProvider struct{}

func (e *ErrorCurrentKeyEncryptProvider) GetCurrent(ctx context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("kek")
}

func (e *ErrorCurrentKeyEncryptProvider) GetByVersion(ctx context.Context, version int) ([]byte, error) {
	return make([]byte, 32), nil
}

func (e *ErrorCurrentKeyEncryptProvider) Rotate(ctx context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("kek")
}

func TestUpdateHandlerUnauthorized(t *testing.T) {
	itemsRepo := &UpdateItemsStoreStub{}
	handler := NewUpdateHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil))
	req := httptest.NewRequest(http.MethodPatch, "/items/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestUpdateHandlerInvalidJSON(t *testing.T) {
	itemsRepo := &UpdateItemsStoreStub{}
	handler := NewUpdateHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	req := testutil.AuthorizedJSONRequest(http.MethodPatch, "/items/"+uuid.New().String(), userID, []byte("{"))
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUpdateHandlerInvalidDataType(t *testing.T) {
	itemsRepo := &UpdateItemsStoreStub{}
	handler := NewUpdateHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 1), jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " t ",
		"data":  json.RawMessage(`{"type":"WRONG"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPatch, "/items/"+uuid.New().String(), userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUpdateHandlerKekErrorOnCurrent(t *testing.T) {
	itemsRepo := &UpdateItemsStoreStub{}
	handler := NewUpdateHandler(itemsRepo, &ErrorCurrentKeyEncryptProvider{}, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " t ",
		"data":  json.RawMessage(`{"type":"TEXT","value":"hello"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPatch, "/items/"+uuid.New().String(), userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestUpdateHandlerDatabaseError(t *testing.T) {
	itemsRepo := &UpdateItemsStoreStub{UpdateError: errors.New("db")}
	handler := NewUpdateHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 7), jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " t ",
		"data":  json.RawMessage(`{"type":"TEXT","value":"hello"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPatch, "/items/"+uuid.New().String(), userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestUpdateHandlerSuccessWithData(t *testing.T) {
	itemsRepo := &UpdateItemsStoreStub{}
	handler := NewUpdateHandler(itemsRepo, keyencrypt.NewStaticProvider(make([]byte, 32), 8), jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " title ",
		"meta":  map[string]string{"k": "v"},
		"data":  json.RawMessage(`{"type":"TEXT","value":"hello"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPatch, "/items/"+uuid.New().String(), userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("unexpected empty body")
	}
}

func TestUpdateHandlerSuccessWithoutDataDecryptsExisting(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kekBytes := make([]byte, 32)
	for i := range kekBytes {
		kekBytes[i] = byte(9 + i%251)
	}
	itemKey := []byte("cccccccccccccccccccccccccccccccc")
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kekBytes, keyAAD, itemKey)
	if err != nil {
		t.Fatalf("encrypt key error: %v", err)
	}
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "TEXT")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAAD, []byte(`{"type":"TEXT","value":"x"}`))
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
		KEKVersion:       9,
		Meta:             map[string]string{"k": "v"},
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	itemsRepo := &UpdateItemsStoreStub{Record: record}
	handler := NewUpdateHandler(itemsRepo, keyencrypt.NewStaticProvider(kekBytes, 9), jwtauth.New("HS256", []byte("secret"), nil))
	body := map[string]interface{}{
		"title": " changed ",
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPatch, "/items/"+itemID.String(), userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, itemID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("unexpected empty body")
	}
}
