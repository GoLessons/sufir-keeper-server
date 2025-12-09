package items

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

func (h *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	var body struct {
		Title *string            `json:"title"`
		Data  *json.RawMessage   `json:"data"`
		Meta  *map[string]string `json:"meta"`
	}
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	upd := repository.ItemUpdateRecord{}
	var respData json.RawMessage
	if body.Title != nil {
		v := strings.TrimSpace(*body.Title)
		upd.Title = &v
	}
	if body.Meta != nil {
		upd.Meta = body.Meta
	}
	if body.Data != nil {
		var dtype struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(*body.Data, &dtype); err != nil || strings.TrimSpace(dtype.Type) == "" {
			httputil.WriteError(w, http.StatusBadRequest, "invalid_data", "Invalid data type")
			return
		}
		disc := strings.TrimSpace(dtype.Type)
		itemDataEncryptionKey := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, itemDataEncryptionKey); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Key generation error")
			return
		}
		dataAAD := []byte(userID.String() + "|" + id.String() + "|" + disc)
		dataNonce, encryptedData, err := aead.Encrypt(itemDataEncryptionKey, dataAAD, *body.Data)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Encrypt error")
			return
		}
		keyEncryptionKey, keyEncryptionKeyVersion, err := h.kek.GetCurrent(r.Context())
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "server_error", "KEK error")
			return
		}
		dataKeyAAD := []byte(userID.String() + "|" + id.String())
		dataKeyNonce, encryptedDataKey, err := aead.Encrypt(keyEncryptionKey, dataKeyAAD, itemDataEncryptionKey)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Encrypt key error")
			return
		}
		upd.DataEncrypted = &encryptedData
		upd.DataNonce = &dataNonce
		upd.DataKeyEncrypted = &encryptedDataKey
		upd.DataKeyNonce = &dataKeyNonce
		upd.KEKVersion = &keyEncryptionKeyVersion
		v := disc
		upd.Type = &v
		respData = *body.Data
	}
	rec, err := h.itemsRepo.Update(r.Context(), userID, id, upd)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	if respData == nil {
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
		respData = json.RawMessage(raw)
	}
	writeItemResponse(w, http.StatusOK, rec, respData)
}
