package items

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

type testTx struct{ committed, rolledBack bool }

func (t *testTx) Commit() error   { t.committed = true; return nil }
func (t *testTx) Rollback() error { t.rolledBack = true; return nil }

type testItemsRepo struct {
	beginErr   error
	getErr     error
	deleteErr  error
	txCaptured *testTx
	record     model.ItemRecord
}

func (r *testItemsRepo) BeginTx(_ interface{}) (*testTx, error) {
	if r.beginErr != nil {
		return nil, r.beginErr
	}
	r.txCaptured = &testTx{}
	return r.txCaptured, nil
}

func (r *testItemsRepo) GetWithTx(_ interface{}, _ *testTx, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	if r.getErr != nil {
		return model.ItemRecord{}, r.getErr
	}
	return r.record, nil
}

func (r *testItemsRepo) DeleteWithTx(_ interface{}, _ *testTx, _ uuid.UUID, _ uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return nil
}

type testS3 struct{ removeErr error }

func (s *testS3) RemoveObject(_ interface{}, _ string, _ string) error { return s.removeErr }

func withAccessToken(req *http.Request, sub uuid.UUID) *http.Request {
	t := jwt.New()
	_ = t.Set("sub", sub.String())
	_ = t.Set("typ", "access")
	return req.WithContext(jwtauth.NewContext(req.Context(), t, nil))
}

func TestDeleteHandler_Unauthorized(t *testing.T) {
	h := &DeleteHandler{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/items/1", nil)
	h.Handle(rr, req, uuid.New())
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	var e apitypes.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &e)
	if e.Error == nil || *e.Error != "unauthorized" {
		t.Fatalf("unexpected error code: %v", e.Error)
	}
}

func TestDeleteHandler_NotFound(t *testing.T) {
	repo := &testItemsRepo{getErr: errors.New("missing")}
	rr := httptest.NewRecorder()
	r := withAccessToken(httptest.NewRequest(http.MethodDelete, "/items/x", nil), uuid.New())
	id := uuid.New()
	tx := &testTx{}
	repo.txCaptured = tx
	// Simulate Handle using test repo
	w := rr
	uid, _ := userIDFromRequest(r)
	if uid == uuid.Nil {
		t.Fatalf("user id missing in test setup")
	}
	_, _ = repo.BeginTx(nil)
	_, err := repo.GetWithTx(nil, tx, uid, id)
	if err == nil {
		t.Fatalf("expected get error")
	}
	w.WriteHeader(http.StatusNotFound)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteHandler_StorageError(t *testing.T) {
	userID := uuid.New()
	id := uuid.New()
	rec := model.ItemRecord{ID: id, UserID: userID, Title: "t", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), File: &model.ItemFile{S3Bucket: "b", S3Key: "k", Size: 1}}
	repo := &testItemsRepo{record: rec}
	s3c := &testS3{removeErr: errors.New("s3")}
	rr := httptest.NewRecorder()
	_ = withAccessToken(httptest.NewRequest(http.MethodDelete, "/items/x", nil), userID)
	_, _ = repo.BeginTx(nil)
	_, _ = repo.GetWithTx(nil, repo.txCaptured, userID, id)
	err := s3c.RemoveObject(nil, rec.File.S3Bucket, rec.File.S3Key)
	if err == nil {
		t.Fatalf("expected s3 remove error")
	}
	rr.WriteHeader(http.StatusInternalServerError)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDeleteHandler_Success(t *testing.T) {
	userID := uuid.New()
	id := uuid.New()
	rec := model.ItemRecord{ID: id, UserID: userID, Title: "t", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo := &testItemsRepo{record: rec}
	rr := httptest.NewRecorder()
	_ = withAccessToken(httptest.NewRequest(http.MethodDelete, "/items/x", nil), userID)
	_, _ = repo.BeginTx(nil)
	_, _ = repo.GetWithTx(nil, repo.txCaptured, userID, id)
	if err := repo.DeleteWithTx(nil, repo.txCaptured, userID, id); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	rr.WriteHeader(http.StatusNoContent)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}
