package items

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
)

func (h *DeleteHandler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	if err := h.itemsRepo.Delete(r.Context(), userID, id); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
