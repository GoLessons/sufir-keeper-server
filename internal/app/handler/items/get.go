package items

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
)

func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	rec, err := h.itemsRepo.Get(r.Context(), userID, id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	keyEncryptionKey, err := h.kek.GetByVersion(r.Context(), rec.KEKVersion)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "KEK error")
		return
	}
	dataKeyAAD := []byte(rec.UserID.String() + "|" + rec.ID.String())
	itemDataEncryptionKey, err := aead.Decrypt(keyEncryptionKey, dataKeyAAD, rec.DataKeyNonce, rec.DataKeyEncrypted)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Decrypt key error")
		return
	}
	dataAAD := []byte(rec.UserID.String() + "|" + rec.ID.String() + "|" + rec.Type)
	raw, err := aead.Decrypt(itemDataEncryptionKey, dataAAD, rec.DataNonce, rec.DataEncrypted)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Decrypt error")
		return
	}
	writeItemResponse(w, http.StatusOK, rec, json.RawMessage(raw))
}
