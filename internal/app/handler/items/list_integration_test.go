package items

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/auth"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestListItemsIntegration(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()

	users := repository.NewUserRepository(dbClient)
	items := repository.NewItemRepository(dbClient)
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)

	userID := uuid.New()
	uniqueLogin := "list_items_user_" + uuid.New().String()
	passwordHash, err := auth.HashPassword("StrongPassword123!")
	require.NoError(t, err)
	userModel := model.NewUser(userID, uniqueLogin, passwordHash, time.Now().UTC())
	_, err = users.Save(t.Context(), userModel)
	require.NoError(t, err)

	handler := NewListHandler(items, kek, tokenAuth)
	params := apitypes.GetItemsParams{}
	lv := 5
	ov := 0
	params.Limit = &lv
	params.Offset = &ov

	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items?limit=5&offset=0", userID, nil)
	rec := httptest.NewRecorder()
	handler.Handle(rec, req, params)
	require.Equal(t, http.StatusOK, rec.Code)
}
