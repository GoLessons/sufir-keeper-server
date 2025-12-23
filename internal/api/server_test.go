package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/minio/sio"
	apiTypes "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestServer_ServiceUnavailableWhenDepsNil(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	s.CreateItem(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("CreateItem expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/items/00000000-0000-0000-0000-000000000000", nil)
	s.GetItem(rec, req, [16]byte{})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GetItem expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/items/00000000-0000-0000-0000-000000000000", nil)
	s.UpdateItem(rec, req, [16]byte{})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("UpdateItem expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/files/presign", nil)
	s.PresignFile(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("PresignFile expected 503, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/files/00000000-0000-0000-0000-000000000000", nil)
	tok := jwt.New()
	_ = tok.Set("sub", "00000000-0000-0000-0000-000000000000")
	_ = tok.Set("typ", "access")
	req = req.WithContext(jwtauth.NewContext(req.Context(), tok, nil))
	s.DownloadFile(rec, req, [16]byte{})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("DownloadFile expected 503, got %d", rec.Code)
	}
}

type apiItemsStoreStub struct {
	ListResult   repository.ListResult
	GetRecord    model.ItemRecord
	CreateRecord model.ItemRecord
	UpdateReturn model.ItemRecord
}

func (s *apiItemsStoreStub) Create(_ context.Context, _ model.ItemRecord) (model.ItemRecord, error) {
	return s.CreateRecord, nil
}

func (s *apiItemsStoreStub) Get(_ context.Context, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	return s.GetRecord, nil
}

func (s *apiItemsStoreStub) List(_ context.Context, _ uuid.UUID, _ *string, _ *string, _ int, _ int) (repository.ListResult, error) {
	return s.ListResult, nil
}

func (s *apiItemsStoreStub) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ repository.ItemUpdateRecord) (model.ItemRecord, error) {
	return s.UpdateReturn, nil
}

func (s *apiItemsStoreStub) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (s *apiItemsStoreStub) BeginTx(_ context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (s *apiItemsStoreStub) GetWithTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ uuid.UUID) (model.ItemRecord, error) {
	return s.GetRecord, nil
}

func (s *apiItemsStoreStub) DeleteWithTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func TestServer_GetItems_Success(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	itemsRepo := &apiItemsStoreStub{
		ListResult: repository.ListResult{
			Items: []model.ItemListRecord{{ID: uuid.New(), Title: "title", Meta: map[string]string{"k": "v"}, CreatedAt: now, UpdatedAt: now}},
			Total: 1,
		},
	}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
		KEKProvider:            keyencrypt.NewStaticProvider(make([]byte, 32), 1),
	})
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items", userID, nil)
	s.GetItems(rec, req, GetItemsParams{})
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Items  []interface{} `json:"items"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.Total)
	require.Equal(t, 20, resp.Limit)
	require.Equal(t, 0, resp.Offset)
	require.Len(t, resp.Items, 1)
}

func TestServer_UploadFile_NotImplemented(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewReader([]byte{}))
	s.UploadFile(rec, req, UploadFileParams{})
	require.Equal(t, http.StatusNotImplemented, rec.Code)
}

func TestServer_PresignFile_Success(t *testing.T) {
	s3stub := &testutil.S3ServiceStub{}
	var _ s3.Service = s3stub
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		S3Client:               s3stub,
	})
	userID := uuid.New()
	fileID := uuid.New()
	body := map[string]interface{}{"filename": "file.bin", "mime": "application/octet-stream", "checksum": "abc", "fileId": fileID.String()}
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, data)
	s.PresignFile(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, s3stub.PresignedCalls, 1)
	require.Equal(t, fileID.String(), s3stub.PresignedCalls[0].Key)
}

func TestServer_PresignFile_ServiceUnavailable(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
	})
	userID := uuid.New()
	body := map[string]interface{}{"filename": "file.bin", "mime": "application/octet-stream", "checksum": "abc", "fileId": uuid.New().String()}
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/files/presign", userID, data)
	s.PresignFile(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestServer_AuthVerifyWrappers(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
	})
	userID := uuid.New()
	rec1 := httptest.NewRecorder()
	req1 := testutil.AuthorizedJSONRequest(http.MethodGet, "/auth-verify", userID, nil)
	s.AuthVerifyGet(rec1, req1)
	require.Equal(t, http.StatusNoContent, rec1.Code)
	require.Equal(t, userID.String(), rec1.Header().Get("X-User-Id"))

	rec2 := httptest.NewRecorder()
	req2 := testutil.AuthorizedJSONRequest(http.MethodPost, "/auth-verify", userID, nil)
	s.AuthVerifyPost(rec2, req2)
	require.Equal(t, http.StatusNoContent, rec2.Code)
	require.Equal(t, userID.String(), rec2.Header().Get("X-User-Id"))
}

func TestServer_RegisterLoginRefreshWrappers(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()
	usersRepository := repository.NewUserRepository(dbClient)
	tokenAuth := jwtauth.New("HS256", []byte("x"), nil)
	s := NewServer(ServerDependencies{
		TokenAuth:              tokenAuth,
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		UsersRepository:        usersRepository,
	})
	login := "api_reg_user_" + uuid.New().String()
	reqBody := map[string]string{"login": login, "password": "StrongPassword123!"}
	bodyBytes, _ := json.Marshal(reqBody)
	recReg := httptest.NewRecorder()
	reqReg := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
	reqReg.Header.Set("Content-Type", "application/json")
	s.RegisterUser(recReg, reqReg)
	require.Equal(t, http.StatusCreated, recReg.Code)

	recLogin := httptest.NewRecorder()
	reqLogin := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(bodyBytes))
	reqLogin.Header.Set("Content-Type", "application/json")
	s.LoginUser(recLogin, reqLogin)
	require.Equal(t, http.StatusOK, recLogin.Code)
	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	require.NoError(t, json.Unmarshal(recLogin.Body.Bytes(), &loginResp))
	require.NotEmpty(t, loginResp.AccessToken)
	require.NotEmpty(t, loginResp.RefreshToken)

	refreshReqBody := map[string]string{"refresh_token": loginResp.RefreshToken}
	refreshBytes, _ := json.Marshal(refreshReqBody)
	recRefresh := httptest.NewRecorder()
	reqRefresh := httptest.NewRequest(http.MethodPatch, "/auth", bytes.NewReader(refreshBytes))
	reqRefresh.Header.Set("Content-Type", "application/json")
	s.RefreshToken(recRefresh, reqRefresh)
	require.Equal(t, http.StatusOK, recRefresh.Code)
}

func TestServer_CreateItem_InvalidJSON(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
		KEKProvider:            keyencrypt.NewStaticProvider(make([]byte, 32), 1),
	})
	userID := uuid.New()
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, []byte("{"))
	s.CreateItem(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestServer_GetItem_Unauthorized(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
		KEKProvider:            keyencrypt.NewStaticProvider(make([]byte, 32), 1),
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items/"+uuid.New().String(), nil)
	s.GetItem(rec, req, apiTypes.UUID(uuid.New()))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestServer_UpdateItem_InvalidDataType(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
		KEKProvider:            keyencrypt.NewStaticProvider(make([]byte, 32), 1),
	})
	userID := uuid.New()
	body := map[string]interface{}{"data": map[string]interface{}{"type": "UNKNOWN"}}
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPut, "/items/"+uuid.New().String(), userID, data)
	s.UpdateItem(rec, req, apiTypes.UUID(uuid.New()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestServer_DeleteItem_NotFound(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()
	itemRepo := repository.NewItemRepository(dbClient)
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemRepo,
		S3Client:               &testutil.S3ServiceStub{},
	})
	userID := uuid.New()
	recorder := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+uuid.New().String(), userID, nil)
	s.DeleteItem(recorder, req, apiTypes.UUID(uuid.New()))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestServer_DownloadFile_MemoryData(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemKey := make([]byte, 32)
	for i := range itemKey {
		itemKey[i] = byte(21 + i%251)
	}
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(31 + i%251)
	}
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, itemKey)
	require.NoError(t, err)
	plaintext := []byte("binary-data")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAAD, plaintext)
	require.NoError(t, err)
	itemsRepo := &apiItemsStoreStub{
		GetRecord: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "bin",
			Type:             "BINARY",
			DataEncrypted:    encData,
			DataNonce:        dataNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       2,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
		KEKProvider:            keyencrypt.NewStaticProvider(kek, 2),
		S3Client:               &testutil.S3ServiceStub{},
	})
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	s.DownloadFile(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, plaintext, rec.Body.Bytes())
}

func TestServer_CreateItem_Success(t *testing.T) {
	userID := uuid.New()
	itemsRepo := &apiItemsStoreStub{
		CreateRecord: model.ItemRecord{
			ID:        uuid.New(),
			UserID:    userID,
			Title:     "note",
			Type:      "TEXT",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
		KEKProvider:            keyencrypt.NewStaticProvider(make([]byte, 32), 1),
	})
	body := map[string]interface{}{"title": "note", "meta": map[string]string{"a": "b"}, "data": map[string]interface{}{"type": "TEXT", "text": "hello"}}
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, data)
	s.CreateItem(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var resp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.ID)
}

func TestServer_CreateItem_ServiceUnavailable(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
	})
	userID := uuid.New()
	body := map[string]interface{}{"title": "note", "data": map[string]interface{}{"type": "TEXT", "text": "x"}}
	data, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/items", userID, data)
	s.CreateItem(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestServer_GetItem_Success(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemKey := make([]byte, 32)
	for i := range itemKey {
		itemKey[i] = byte(7 + i%251)
	}
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(11 + i%251)
	}
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, itemKey)
	require.NoError(t, err)
	data := json.RawMessage([]byte(`{"type":"TEXT","text":"hello"}`))
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "TEXT")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAAD, data)
	require.NoError(t, err)
	itemsRepo := &apiItemsStoreStub{
		GetRecord: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "note",
			Type:             "TEXT",
			DataEncrypted:    encData,
			DataNonce:        dataNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       3,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
		KEKProvider:            keyencrypt.NewStaticProvider(kek, 3),
	})
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	s.GetItem(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "TEXT", resp.Data["type"])
}

func TestServer_GetItem_ServiceUnavailable(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
	})
	userID := uuid.New()
	itemID := uuid.New()
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items/"+itemID.String(), userID, nil)
	s.GetItem(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestServer_UpdateItem_TitleOnly(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	itemKey := make([]byte, 32)
	for i := range itemKey {
		itemKey[i] = byte(5 + i%251)
	}
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(13 + i%251)
	}
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, itemKey)
	require.NoError(t, err)
	data := json.RawMessage([]byte(`{"type":"TEXT","text":"hello"}`))
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "TEXT")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAAD, data)
	require.NoError(t, err)
	itemsRepo := &apiItemsStoreStub{
		UpdateReturn: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "updated",
			Type:             "TEXT",
			DataEncrypted:    encData,
			DataNonce:        dataNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       9,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
		KEKProvider:            keyencrypt.NewStaticProvider(kek, 9),
	})
	body := map[string]interface{}{"title": "updated"}
	dataBytes, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPut, "/items/"+itemID.String(), userID, dataBytes)
	s.UpdateItem(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data  map[string]interface{} `json:"data"`
		Title string                 `json:"title"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "updated", resp.Title)
	require.Equal(t, "TEXT", resp.Data["type"])
}

func TestServer_UpdateItem_ServiceUnavailable(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
	})
	userID := uuid.New()
	itemID := uuid.New()
	body := map[string]interface{}{"title": "updated"}
	dataBytes, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodPut, "/items/"+itemID.String(), userID, dataBytes)
	s.UpdateItem(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestServer_DownloadFile_ServiceUnavailable(t *testing.T) {
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        &apiItemsStoreStub{},
	})
	userID := uuid.New()
	itemID := uuid.New()
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	s.DownloadFile(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestServer_DeleteItem_RemovesS3(t *testing.T) {
	dbClient := testutil.CreateDatabaseClientForIntegrationTests(t)
	defer func() { _ = dbClient.Close() }()
	userRepo := repository.NewUserRepository(dbClient)
	itemRepo := repository.NewItemRepository(dbClient)
	userID := uuid.New()
	login := "api_delete_user_" + uuid.New().String()
	_, err := userRepo.Save(t.Context(), model.NewUser(userID, login, "hash", time.Now().UTC()))
	require.NoError(t, err)
	itemID := uuid.New()
	rec := model.ItemRecord{
		ID:               itemID,
		UserID:           userID,
		Title:            "binary",
		Type:             "BINARY",
		DataEncrypted:    []byte("de"),
		DataNonce:        []byte("dn"),
		DataKeyEncrypted: []byte("ke"),
		DataKeyNonce:     []byte("kn"),
		KEKVersion:       1,
		Meta:             map[string]string{"mime": "application/octet-stream"},
		File:             &model.ItemFile{S3Bucket: "protected", S3Key: "objects/" + itemID.String(), Size: 10, SHA256: "abc"},
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	_, err = itemRepo.Create(t.Context(), rec)
	require.NoError(t, err)
	s3stub := &testutil.S3ServiceStub{}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemRepo,
		S3Client:               s3stub,
	})
	recorder := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodDelete, "/items/"+itemID.String(), userID, nil)
	s.DeleteItem(recorder, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusNoContent, recorder.Code)
	found := false
	for _, r := range s3stub.RemovedRecords {
		if r.Bucket == "protected" && r.Key == "objects/"+itemID.String() {
			found = true
			break
		}
	}
	require.True(t, found)
	_, err = itemRepo.Get(t.Context(), userID, itemID)
	require.Error(t, err)
}

func TestServer_DownloadFile_StreamSuccess(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(1 + i%251)
	}
	itemKey := make([]byte, 32)
	for i := range itemKey {
		itemKey[i] = byte(2 + i%251)
	}
	keyAAD := []byte(userID.String() + "|" + itemID.String())
	keyNonce, encKey, err := aead.Encrypt(kek, keyAAD, itemKey)
	require.NoError(t, err)
	plaintext := []byte("file-bytes")
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + "BINARY")
	dataNonce, encData, err := aead.Encrypt(itemKey, dataAAD, plaintext)
	require.NoError(t, err)
	itemsRepo := &apiItemsStoreStub{
		GetRecord: model.ItemRecord{
			ID:               itemID,
			UserID:           userID,
			Title:            "f.bin",
			Type:             "BINARY",
			DataEncrypted:    encData,
			DataNonce:        dataNonce,
			DataKeyEncrypted: encKey,
			DataKeyNonce:     keyNonce,
			KEKVersion:       7,
			Meta:             map[string]string{"mime": "application/octet-stream"},
			File:             &model.ItemFile{S3Bucket: "protected", S3Key: "key", Size: int64(len(plaintext))},
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}
	encReader, err := sio.EncryptReader(bytes.NewReader(plaintext), sio.Config{
		Key:          itemKey,
		MinVersion:   sio.Version20,
		CipherSuites: []byte{sio.AES_256_GCM},
	})
	require.NoError(t, err)
	encryptedBytes := make([]byte, 0, len(plaintext)+1024)
	buf := make([]byte, 256)
	for {
		n, er := encReader.Read(buf)
		if n > 0 {
			encryptedBytes = append(encryptedBytes, buf[:n]...)
		}
		if er != nil {
			break
		}
	}
	s3stub := &testutil.S3ServiceStub{GetObjectData: encryptedBytes, StatSize: int64(len(plaintext))}
	s := NewServer(ServerDependencies{
		TokenAuth:              jwtauth.New("HS256", []byte("x"), nil),
		AccessTokenTTLSeconds:  3600,
		RefreshTokenTTLSeconds: 3600,
		ItemsRepository:        itemsRepo,
		KEKProvider:            keyencrypt.NewStaticProvider(kek, 7),
		S3Client:               s3stub,
	})
	rec := httptest.NewRecorder()
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/files/"+itemID.String(), userID, nil)
	s.DownloadFile(rec, req, apiTypes.UUID(itemID))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, plaintext, rec.Body.Bytes())
}
