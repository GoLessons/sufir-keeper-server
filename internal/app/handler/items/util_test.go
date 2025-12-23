package items

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

func TestDefaultLimitAndOffset(t *testing.T) {
	if v := defaultLimit(nil); v != 20 {
		t.Fatalf("unexpected default limit: %d", v)
	}
	l := 0
	if v := defaultLimit(&l); v != 20 {
		t.Fatalf("unexpected limit for 0: %d", v)
	}
	l = 200
	if v := defaultLimit(&l); v != 100 {
		t.Fatalf("unexpected limit clamp: %d", v)
	}
	l = 50
	if v := defaultLimit(&l); v != 50 {
		t.Fatalf("unexpected limit pass-through: %d", v)
	}

	if v := defaultOffset(nil); v != 0 {
		t.Fatalf("unexpected default offset: %d", v)
	}
	o := -1
	if v := defaultOffset(&o); v != 0 {
		t.Fatalf("unexpected offset negative: %d", v)
	}
	o = 10
	if v := defaultOffset(&o); v != 10 {
		t.Fatalf("unexpected offset pass-through: %d", v)
	}
}

func TestUserIDFromRequest(t *testing.T) {
	id := uuid.New()
	tok := jwt.New()
	_ = tok.Set("sub", id.String())
	_ = tok.Set("typ", "access")
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rctx := jwtauth.NewContext(req.Context(), tok, nil)
	req = req.WithContext(rctx)
	got, ok := userIDFromRequest(req)
	if !ok {
		t.Fatalf("expected ok")
	}
	if got != id {
		t.Fatalf("unexpected id: %s", got.String())
	}
}

func TestWriteItemResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	uid := uuid.New()
	id := uuid.New()
	title := "t"
	meta := map[string]string{"k": "v"}
	body := struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}{Type: "TEXT", Value: "hello"}
	dataBytes, _ := json.Marshal(body)
	record := model.ItemRecord{ID: id, UserID: uid, Title: title, Meta: meta}
	writeItemResponse(rec, http.StatusCreated, record, json.RawMessage(dataBytes))
	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var got apitypes.ItemResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Id == nil || *got.Id != openapi_types.UUID(id) {
		t.Fatalf("unexpected id")
	}
	if got.UserId == nil || *got.UserId != openapi_types.UUID(uid) {
		t.Fatalf("unexpected user id")
	}
	if got.Title == nil || *got.Title != title {
		t.Fatalf("unexpected title")
	}
	if got.Meta == nil || (*got.Meta)["k"] != "v" {
		t.Fatalf("unexpected meta")
	}
	if got.Data == nil {
		t.Fatalf("expected data union")
	}
}

func TestMakeItemListJSON(t *testing.T) {
	uid := uuid.New()
	id := uuid.New()
	title := "t"
	meta := map[string]string{"k": "v"}
	record := model.ItemListRecord{ID: id, Title: title, Meta: meta}
	resp := makeItemListJSON(record)
	if resp.Id == nil || *resp.Id != openapi_types.UUID(id) {
		t.Fatalf("unexpected id")
	}
	if resp.Title == nil || *resp.Title != title {
		t.Fatalf("unexpected title")
	}
	if resp.Meta == nil || (*resp.Meta)["k"] != "v" {
		t.Fatalf("unexpected meta")
	}
	_ = uid
}
