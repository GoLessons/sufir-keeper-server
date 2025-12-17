package items

import (
	"encoding/json"
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

func TestGetItemIntegration(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()

	users := repository.NewUserRepository(dbClient)
	items := repository.NewItemRepository(dbClient)
	kek := keyencrypt.NewStaticProvider(make([]byte, 32), 1)
	tokenAuth := jwtauth.New("HS256", []byte("integration-secret"), nil)

	userID := uuid.New()
	uniqueLogin := "get_item_user_" + uuid.New().String()
	passwordHash, err := auth.HashPassword("StrongPassword123!")
	require.NoError(t, err)
	userModel := model.NewUser(userID, uniqueLogin, passwordHash, time.Now().UTC())
	_, err = users.Save(t.Context(), userModel)
	require.NoError(t, err)

	// create item
	createHandler := NewCreateHandler(items, kek, tokenAuth)
	var data apitypes.ItemCreate_Data
	require.NoError(t, data.FromTextData(apitypes.TextData{Type: "TEXT", Value: "Hello"}))
	reqBody := apitypes.ItemCreate{Title: "Title", Data: data}
	bodyBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)
	createReq := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, bodyBytes)
	createRec := httptest.NewRecorder()
	createHandler.Handle(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created apitypes.ItemResponse
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	itemID := uuid.UUID(*created.Id)

	// get
	getHandler := NewGetHandler(items, kek, tokenAuth)
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	rec := httptest.NewRecorder()
	getHandler.Handle(rec, req, itemID)
	require.Equal(t, http.StatusOK, rec.Code)
}
