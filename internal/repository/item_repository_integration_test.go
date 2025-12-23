package repository

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func createUserForRepositoryTests(t *testing.T, client *db.Client, loginSuffix string) uuid.UUID {
	users := NewUserRepository(client)
	userID := uuid.New()
	login := "repo_user_" + loginSuffix
	_, err := users.Save(t.Context(), model.NewUser(userID, login, "hash", time.Now().UTC()))
	require.NoError(t, err)
	return userID
}

func TestItemRepositoryCreateAndGetWithFile(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)

	userID := createUserForRepositoryTests(t, client, uuid.New().String())
	itemID := uuid.New()
	rec := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "file-item",
		Type:             "BINARY",
		DataEncrypted:    []byte("de"),
		DataNonce:        []byte("dn"),
		DataKeyEncrypted: []byte("ke"),
		DataKeyNonce:     []byte("kn"),
		KEKVersion:       1,
		Meta:             map[string]string{"mime": "application/octet-stream"},
		File:             &model.ItemFile{S3Bucket: "protected", S3Key: "objects/" + itemID.String(), Size: 1234, SHA256: "abc"},
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	_, err := repo.Create(t.Context(), rec)
	require.NoError(t, err)

	got, err := repo.Get(t.Context(), userID, itemID)
	require.NoError(t, err)
	require.Equal(t, itemID, got.ID)
	require.NotNil(t, got.File)
	require.Equal(t, "protected", got.File.S3Bucket)
	require.Equal(t, "objects/"+itemID.String(), got.File.S3Key)
	require.Equal(t, int64(1234), got.File.Size)
	require.Equal(t, "abc", got.File.SHA256)
	require.Equal(t, "application/octet-stream", got.Meta["mime"])
}

func TestItemRepositoryListFilterAndSearch(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)

	userID := createUserForRepositoryTests(t, client, uuid.New().String())
	titles := []string{"alpha", "beta", "alphabet"}
	for _, title := range titles {
		itemID := uuid.New()
		rec := model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            title,
			Type:             "TEXT",
			DataEncrypted:    []byte("de"),
			DataNonce:        []byte("dn"),
			DataKeyEncrypted: []byte("ke"),
			DataKeyNonce:     []byte("kn"),
			KEKVersion:       1,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
		_, err := repo.Create(t.Context(), rec)
		require.NoError(t, err)
	}
	filterType := "TEXT"
	search := "alpha"
	result, err := repo.List(t.Context(), userID, &filterType, &search, 10, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.Total, 2)
	require.GreaterOrEqual(t, len(result.Items), 2)
}

func TestItemRepositoryUpdateMetaNilAndEmpty(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)

	userID := createUserForRepositoryTests(t, client, uuid.New().String())
	itemID := uuid.New()
	rec := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "meta-item",
		Type:             "TEXT",
		DataEncrypted:    []byte("de"),
		DataNonce:        []byte("dn"),
		DataKeyEncrypted: []byte("ke"),
		DataKeyNonce:     []byte("kn"),
		KEKVersion:       1,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Meta:             map[string]string{"a": "b"},
	}
	_, err := repo.Create(t.Context(), rec)
	require.NoError(t, err)

	var nilMeta map[string]string
	updNil := ItemUpdateRecord{Meta: &nilMeta}
	_, err = repo.Update(t.Context(), userID, itemID, updNil)
	require.NoError(t, err)
	gotNil, err := repo.Get(t.Context(), userID, itemID)
	require.NoError(t, err)
	require.Nil(t, gotNil.Meta)

	emptyMeta := map[string]string{}
	updEmpty := ItemUpdateRecord{Meta: &emptyMeta}
	_, err = repo.Update(t.Context(), userID, itemID, updEmpty)
	require.NoError(t, err)
	gotEmpty, err := repo.Get(t.Context(), userID, itemID)
	require.NoError(t, err)
	require.NotNil(t, gotEmpty.Meta)
	require.Len(t, gotEmpty.Meta, 0)
}

func TestItemRepositoryUpdateTitleAndType(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)
	userID := createUserForRepositoryTests(t, client, uuid.New().String())
	itemID := uuid.New()
	rec := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "original",
		Type:             "TEXT",
		DataEncrypted:    []byte("de"),
		DataNonce:        []byte("dn"),
		DataKeyEncrypted: []byte("ke"),
		DataKeyNonce:     []byte("kn"),
		KEKVersion:       1,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	_, err := repo.Create(t.Context(), rec)
	require.NoError(t, err)
	newTitle := "updated"
	newType := "BINARY"
	upd := ItemUpdateRecord{Title: &newTitle, Type: &newType}
	res, err := repo.Update(t.Context(), userID, itemID, upd)
	require.NoError(t, err)
	require.Equal(t, newTitle, res.Title)
	require.Equal(t, newType, res.Type)
}

func TestItemRepositoryListWithLimitAndOffset(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)
	userID := createUserForRepositoryTests(t, client, uuid.New().String())
	for i := 0; i < 25; i++ {
		itemID := uuid.New()
		rec := model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "t-" + strconv.Itoa(i),
			Type:             "TEXT",
			DataEncrypted:    []byte("de"),
			DataNonce:        []byte("dn"),
			DataKeyEncrypted: []byte("ke"),
			DataKeyNonce:     []byte("kn"),
			KEKVersion:       1,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
		_, err := repo.Create(t.Context(), rec)
		require.NoError(t, err)
	}
	filterType := "TEXT"
	limit := 10
	offset := 5
	result, err := repo.List(t.Context(), userID, &filterType, nil, limit, offset)
	require.NoError(t, err)
	require.Equal(t, 25, result.Total)
	require.Equal(t, limit, len(result.Items))
}

func TestItemRepositoryDeleteNotFound(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)
	err := repo.Delete(t.Context(), uuid.New(), uuid.New())
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestItemRepositoryTxGetAndDelete(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewItemRepository(client)

	userID := createUserForRepositoryTests(t, client, uuid.New().String())
	itemID := uuid.New()
	rec := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "tx-item",
		Type:             "TEXT",
		DataEncrypted:    []byte("de"),
		DataNonce:        []byte("dn"),
		DataKeyEncrypted: []byte("ke"),
		DataKeyNonce:     []byte("kn"),
		KEKVersion:       1,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	_, err := repo.Create(t.Context(), rec)
	require.NoError(t, err)

	tx, err := repo.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	got, err := repo.GetWithTx(t.Context(), tx, userID, itemID)
	require.NoError(t, err)
	require.Equal(t, itemID, got.ID)

	err = repo.DeleteWithTx(t.Context(), tx, userID, itemID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	_, err = repo.Get(t.Context(), userID, itemID)
	require.Error(t, err)
}
