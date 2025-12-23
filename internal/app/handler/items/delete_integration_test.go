package items

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/auth"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestDeleteItemIntegration(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()

	users := repository.NewUserRepository(dbClient)
	items := repository.NewItemRepository(dbClient)
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 2)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret-2"), nil)

	passwordHash, err := auth.HashPassword("StrongPassword123!")
	require.NoError(t, err)
	userID := uuid.New()
	uniqueLogin := "delete_item_user_" + uuid.New().String()
	userModel := model.NewUser(userID, uniqueLogin, passwordHash, time.Now().UTC())
	_, err = users.Save(t.Context(), userModel)
	require.NoError(t, err)

	itemID := uuid.New()
	keyEncryptionKey, keyVersion, err := kek.GetCurrent(t.Context())
	require.NoError(t, err)
	dataEncryptionKey := make([]byte, 32)
	_, err = rand.Read(dataEncryptionKey)
	require.NoError(t, err)
	dataKeyAAD := []byte(userID.String() + "|" + itemID.String())
	dataKeyNonce, encryptedDataKey, err := aead.Encrypt(keyEncryptionKey, dataKeyAAD, dataEncryptionKey)
	require.NoError(t, err)

	itemRecord := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            itemID.String(),
		Type:             "BINARY",
		DataKeyEncrypted: encryptedDataKey,
		DataKeyNonce:     dataKeyNonce,
		KEKVersion:       keyVersion,
		File: &model.ItemFile{
			S3Bucket: "keeper-protected",
			S3Key:    "uploads/" + itemID.String(),
			Size:     1024,
			SHA256:   "",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err = items.Create(t.Context(), itemRecord)
	require.NoError(t, err)

	fakeS3 := &testutil.S3ServiceStub{}
	deleteHandler := NewDeleteHandler(items, kek, tokenAuth, fakeS3)

	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+itemID.String(), userID, nil)
	deleteHandler.Handle(rec, req, itemID)
	require.Equal(t, http.StatusNoContent, rec.Code)

	require.Len(t, fakeS3.RemovedRecords, 1)
	require.Equal(t, "keeper-protected", fakeS3.RemovedRecords[0].Bucket)
	require.Equal(t, "uploads/"+itemID.String(), fakeS3.RemovedRecords[0].Key)
}
