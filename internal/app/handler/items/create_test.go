package items

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

type CreateItemsStoreStub struct {
	Err    error
	Record model.ItemRecord
}

func (s *CreateItemsStoreStub) Create(ctx context.Context, rec model.ItemRecord) (model.ItemRecord, error) {
	if s.Err != nil {
		return model.ItemRecord{}, s.Err
	}
	return rec, nil
}

func (s *CreateItemsStoreStub) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *CreateItemsStoreStub) List(ctx context.Context, userID uuid.UUID, filterType *string, search *string, limit int, offset int) (repository.ListResult, error) {
	return repository.ListResult{}, errors.New("not implemented")
}

func (s *CreateItemsStoreStub) Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, upd repository.ItemUpdateRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *CreateItemsStoreStub) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (s *CreateItemsStoreStub) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (s *CreateItemsStoreStub) GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *CreateItemsStoreStub) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

type ErrorKeyEncryptProvider struct{}

func (e *ErrorKeyEncryptProvider) GetCurrent(ctx context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("kek")
}

func (e *ErrorKeyEncryptProvider) GetByVersion(ctx context.Context, version int) ([]byte, error) {
	return nil, errors.New("kek")
}

func (e *ErrorKeyEncryptProvider) Rotate(ctx context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("kek")
}

func TestCreateHandlerUnauthorized(t *testing.T) {
	itemsRepo := &CreateItemsStoreStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewCreateHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestCreateHandlerInvalidJSON(t *testing.T) {
	itemsRepo := &CreateItemsStoreStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewCreateHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, []byte("{"))
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateHandlerInvalidDataType(t *testing.T) {
	itemsRepo := &CreateItemsStoreStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewCreateHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " t ",
		"data":  json.RawMessage(`{"type":"WRONG"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateHandlerKekError(t *testing.T) {
	itemsRepo := &CreateItemsStoreStub{}
	handler := NewCreateHandler(itemsRepo, &ErrorKeyEncryptProvider{}, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " t ",
		"data":  json.RawMessage(`{"type":"TEXT","value":"hello"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestCreateHandlerDatabaseError(t *testing.T) {
	itemsRepo := &CreateItemsStoreStub{Err: errors.New("db")}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	handler := NewCreateHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " t ",
		"data":  json.RawMessage(`{"type":"TEXT","value":"hello"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestCreateHandlerSuccess(t *testing.T) {
	itemsRepo := &CreateItemsStoreStub{}
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 3)
	handler := NewCreateHandler(itemsRepo, kek, jwtauth.New("HS256", []byte("secret"), nil))
	userID := uuid.New()
	body := map[string]interface{}{
		"title": " title ",
		"meta":  map[string]string{"k": "v"},
		"data":  json.RawMessage(`{"type":"TEXT","value":"hello"}`),
	}
	buf, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, buf)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("unexpected empty body")
	}
}
