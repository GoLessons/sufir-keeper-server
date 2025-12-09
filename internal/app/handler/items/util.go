package items

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

func userIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil || claims == nil {
		return uuid.Nil, false
	}
	sub, _ := claims["sub"].(string)
	id, err := uuid.Parse(strings.TrimSpace(sub))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// контекстные костыли удалены; параметры передаются напрямую в методы Handle

func defaultLimit(v *int) int {
	if v == nil || *v <= 0 {
		return 20
	}
	if *v > 100 {
		return 100
	}
	return *v
}

func defaultOffset(v *int) int {
	if v == nil || *v < 0 {
		return 0
	}
	return *v
}

func writeItemResponse(w http.ResponseWriter, status int, rec model.ItemRecord, data json.RawMessage) {
	id := openapi_types.UUID(rec.ID)
	uid := openapi_types.UUID(rec.UserID)
	title := rec.Title
	meta := rec.Meta
	created := rec.CreatedAt
	updated := rec.UpdatedAt
	resp := apitypes.ItemResponse{Id: &id, UserId: &uid, Title: &title, Meta: &meta, CreatedAt: &created, UpdatedAt: &updated}
	if len(data) > 0 {
		var union apitypes.ItemResponse_Data
		_ = union.UnmarshalJSON(data)
		resp.Data = &union
	}
	httputil.WriteJSON(w, status, resp)
}

func makeItemListJSON(rec model.ItemListRecord) apitypes.ItemListResponse {
	id := openapi_types.UUID(rec.ID)
	title := rec.Title
	meta := rec.Meta
	created := rec.CreatedAt
	updated := rec.UpdatedAt
	return apitypes.ItemListResponse{Id: &id, Title: &title, Meta: &meta, CreatedAt: &created, UpdatedAt: &updated}
}
