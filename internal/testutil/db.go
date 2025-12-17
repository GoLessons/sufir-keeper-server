package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/db"
)

const DefaultIntegrationPostgresDataSourceName = "postgres://keeper:keeper@postgres:5432/keeper?sslmode=disable"

func CreateDatabaseClientForIntegrationTests(t *testing.T) *db.Client {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = DefaultIntegrationPostgresDataSourceName
	}
	client, err := db.NewClient(t.Context(), dsn, db.Options{})
	require.NoError(t, err)
	return client
}
