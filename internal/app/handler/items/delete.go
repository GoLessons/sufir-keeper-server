package items

import (
	"database/sql"
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
	tx, err := h.itemsRepo.BeginTx(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	rec, err := h.itemsRepo.GetWithTx(r.Context(), tx, userID, id)
	if err != nil {
		_ = tx.Rollback()
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	if rec.File != nil && h.s3client != nil {
		if err := h.s3client.RemoveObject(r.Context(), rec.File.S3Bucket, rec.File.S3Key); err != nil {
			_ = tx.Rollback()
			httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Storage error")
			return
		}
	}
	if err := h.itemsRepo.DeleteWithTx(r.Context(), tx, userID, id); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "Item not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
