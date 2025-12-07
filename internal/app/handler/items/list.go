package items

import (
	"net/http"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
)

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request, filterType *string, search *string, limit int, offset int) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	res, err := h.itemsRepo.List(r.Context(), userID, filterType, search, defaultLimit(&limit), defaultOffset(&offset))
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	out := struct {
		Items  []apitypes.ItemListResponse `json:"items"`
		Total  int                         `json:"total"`
		Limit  int                         `json:"limit"`
		Offset int                         `json:"offset"`
	}{Items: make([]apitypes.ItemListResponse, 0, len(res.Items)), Total: res.Total, Limit: defaultLimit(&limit), Offset: defaultOffset(&offset)}
	for _, it := range res.Items {
		out.Items = append(out.Items, makeItemListJSON(it))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}
