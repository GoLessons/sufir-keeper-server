package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestUserRepositorySaveGetAndVersions(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewUserRepository(client)

	userID := uuid.New()
	login := "user_repo_" + uuid.New().String()
	_, err := repo.Save(t.Context(), model.NewUser(userID, login, "hash", time.Now().UTC()))
	require.NoError(t, err)

	rec, err := repo.GetByLogin(t.Context(), login)
	require.NoError(t, err)
	require.Equal(t, userID, rec.ID)
	require.Equal(t, login, rec.Login)

	version, err := repo.EnsureRefreshVersion(t.Context(), userID)
	require.NoError(t, err)
	require.Equal(t, 1, version)

	newVersion, err := repo.IncrementRefreshVersion(t.Context(), userID)
	require.NoError(t, err)
	require.Equal(t, 2, newVersion)
}

func TestUserRepositoryUpdateOnConflict(t *testing.T) {
	client := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = client.Close() }()
	repo := NewUserRepository(client)

	userID := uuid.New()
	login := "user_repo_conflict_" + uuid.New().String()
	_, err := repo.Save(t.Context(), model.NewUser(userID, login, "hash", time.Now().UTC()))
	require.NoError(t, err)

	newLogin := login + "_updated"
	_, err = repo.Save(t.Context(), model.NewUser(userID, newLogin, "hash2", time.Now().UTC()))
	require.NoError(t, err)

	rec, err := repo.GetByLogin(t.Context(), newLogin)
	require.NoError(t, err)
	require.Equal(t, userID, rec.ID)
	require.Equal(t, newLogin, rec.Login)

	version, err := repo.GetRefreshVersion(t.Context(), userID)
	require.NoError(t, err)
	require.Equal(t, 1, version)
}
