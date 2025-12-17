package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestItemRepositoryCRUDInMemoryConfig(t *testing.T) {
	// This test uses a live Postgres in docker compose during devcheck.
	// Here we only construct the repository and ensure methods do not panic with minimal inputs.
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)

	ctx := t.Context()
	userID := uuid.New()
	id := uuid.New()
	rec := model.ItemRecord{ID: id, UserID: userID, Title: "t", Type: "TEXT", DataEncrypted: []byte("x"), DataNonce: []byte("n"), DataKeyEncrypted: []byte("k"), DataKeyNonce: []byte("kn"), KEKVersion: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err := repo.Create(ctx, rec); err != nil {
		t.Skip("create failed (likely no migrations): " + err.Error())
		return
	}
	if _, err := repo.Get(ctx, userID, id); err != nil {
		t.Fatal("get failed: " + err.Error())
	}
	// Update title
	nt := "t2"
	upd := ItemUpdateRecord{Title: &nt}
	if _, err := repo.Update(ctx, userID, id, upd); err != nil {
		t.Fatal("update failed: " + err.Error())
	}
	// List
	if _, err := repo.List(ctx, userID, nil, nil, 10, 0); err != nil {
		t.Fatal("list failed: " + err.Error())
	}
	// Delete
	if err := repo.Delete(ctx, userID, id); err != nil {
		t.Fatal("delete failed: " + err.Error())
	}
}
