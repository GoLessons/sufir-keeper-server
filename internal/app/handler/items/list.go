package items

import (
	"net/http"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
)

func (h *ListHandler) Handle(w http.ResponseWriter, r *http.Request, params apitypes.GetItemsParams) {
	var filterType *string
	if params.Type != nil {
		v := string(*params.Type)
		filterType = &v
	}

	userID, ok := userIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	limitValue := defaultLimit(params.Limit)
	offsetValue := defaultOffset(params.Offset)
	listResult, err := h.itemsRepo.List(r.Context(), userID, filterType, params.S, limitValue, offsetValue)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	responseOutput := struct {
		Items  []apitypes.ItemListResponse `json:"items"`
		Total  int                         `json:"total"`
		Limit  int                         `json:"limit"`
		Offset int                         `json:"offset"`
	}{Items: make([]apitypes.ItemListResponse, 0, len(listResult.Items)), Total: listResult.Total, Limit: limitValue, Offset: offsetValue}
	for _, itemRecord := range listResult.Items {
		responseOutput.Items = append(responseOutput.Items, makeItemListJSON(itemRecord))
	}
	httputil.WriteJSON(w, http.StatusOK, responseOutput)
}
