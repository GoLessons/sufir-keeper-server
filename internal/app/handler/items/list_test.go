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

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

type ListItemsStoreStub struct {
	Err    error
	Result repository.ListResult
}

func TestListHandlerRepositoryError(t *testing.T) {
	itemsRepo := &ListItemsStoreStub{Err: errors.New("db")}
	handler := NewListHandler(itemsRepo, &NoopKeyEncryptProvider{}, jwtauth.New("HS256", []byte("secret"), nil))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items", uuid.New(), nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, apitypes.GetItemsParams{})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func (s *ListItemsStoreStub) Create(ctx context.Context, rec model.ItemRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *ListItemsStoreStub) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *ListItemsStoreStub) List(ctx context.Context, userID uuid.UUID, filterType *string, search *string, limit int, offset int) (repository.ListResult, error) {
	if s.Err != nil {
		return repository.ListResult{}, s.Err
	}
	return s.Result, nil
}

func (s *ListItemsStoreStub) Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, upd repository.ItemUpdateRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *ListItemsStoreStub) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (s *ListItemsStoreStub) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (s *ListItemsStoreStub) GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, errors.New("not implemented")
}

func (s *ListItemsStoreStub) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error {
	return errors.New("not implemented")
}

type NoopKeyEncryptProvider struct{}

func (n *NoopKeyEncryptProvider) GetCurrent(ctx context.Context) ([]byte, int, error) {
	return make([]byte, 32), 1, nil
}

func (n *NoopKeyEncryptProvider) GetByVersion(ctx context.Context, version int) ([]byte, error) {
	return make([]byte, 32), nil
}

func (n *NoopKeyEncryptProvider) Rotate(ctx context.Context) ([]byte, int, error) {
	return make([]byte, 32), 2, nil
}

func TestListHandlerUnauthorized(t *testing.T) {
	itemsRepo := &ListItemsStoreStub{}
	handler := NewListHandler(itemsRepo, &NoopKeyEncryptProvider{}, jwtauth.New("HS256", []byte("secret"), nil))
	params := apitypes.GetItemsParams{}
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, params)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestListHandlerSuccess(t *testing.T) {
	now := time.Now().UTC()
	items := []model.ItemListRecord{
		{ID: uuid.New(), Title: "a", CreatedAt: now.Add(-time.Hour), UpdatedAt: now, Meta: map[string]string{"k": "v"}},
		{ID: uuid.New(), Title: "b", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now, Meta: map[string]string{"x": "y"}},
	}
	itemsRepo := &ListItemsStoreStub{Result: repository.ListResult{Items: items, Total: 2}}
	handler := NewListHandler(itemsRepo, &NoopKeyEncryptProvider{}, jwtauth.New("HS256", []byte("secret"), nil))
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items?limit=20", uuid.New(), nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req, apitypes.GetItemsParams{Limit: func(i int) *int { return &i }(20)})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("unexpected empty body")
	}
}
