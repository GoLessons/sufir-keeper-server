package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

type itemsListOnlyStub struct{ Result repository.ListResult }

func (s *itemsListOnlyStub) Create(_ context.Context, _ model.ItemRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, nil
}

func (s *itemsListOnlyStub) Get(_ context.Context, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, nil
}

func (s *itemsListOnlyStub) List(_ context.Context, _ uuid.UUID, _ *string, _ *string, _ int, _ int) (repository.ListResult, error) {
	return s.Result, nil
}

func (s *itemsListOnlyStub) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ repository.ItemUpdateRecord) (model.ItemRecord, error) {
	return model.ItemRecord{}, nil
}
func (s *itemsListOnlyStub) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }
func (s *itemsListOnlyStub) BeginTx(_ context.Context) (*sql.Tx, error)               { return nil, nil }
func (s *itemsListOnlyStub) GetWithTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	return model.ItemRecord{}, nil
}

func (s *itemsListOnlyStub) DeleteWithTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func TestServer_GetItems_OnlyList(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	itemsRepo := &itemsListOnlyStub{Result: repository.ListResult{
		Items: []model.ItemListRecord{{ID: uuid.New(), Title: "alpha", CreatedAt: now, UpdatedAt: now}},
		Total: 1,
	}}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
	})
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items", userID, nil)
	s.GetItems(rec, req, GetItemsParams{})
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Items []interface{} `json:"items"`
		Total int           `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
}
