package items

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	var body struct {
		Meta  *map[string]string `json:"meta"`
		Title string             `json:"title"`
		Data  json.RawMessage    `json:"data"`
	}
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	var dtype struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body.Data, &dtype); err != nil || strings.TrimSpace(dtype.Type) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_data", "Invalid data type")
		return
	}
	disc := strings.TrimSpace(dtype.Type)
	itemID := uuid.New()
	itemDataEncryptionKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, itemDataEncryptionKey); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Key generation error")
		return
	}
	dataAAD := []byte(userID.String() + "|" + itemID.String() + "|" + disc)
	dataNonce, encryptedData, err := aead.Encrypt(itemDataEncryptionKey, dataAAD, body.Data)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Encrypt error")
		return
	}
	keyEncryptionKey, keyEncryptionKeyVersion, err := h.kek.GetCurrent(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "KEK error")
		return
	}
	dataKeyAAD := []byte(userID.String() + "|" + itemID.String())
	dataKeyNonce, encryptedDataKey, err := aead.Encrypt(keyEncryptionKey, dataKeyAAD, itemDataEncryptionKey)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Encrypt key error")
		return
	}
	rec := model.ItemRecord{ID: itemID, UserID: userID, Title: strings.TrimSpace(body.Title), Type: disc, DataEncrypted: encryptedData, DataNonce: dataNonce, DataKeyEncrypted: encryptedDataKey, DataKeyNonce: dataKeyNonce, KEKVersion: keyEncryptionKeyVersion, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if body.Meta != nil {
		rec.Meta = *body.Meta
	}
	saved, err := h.itemsRepo.Create(r.Context(), rec)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	writeItemResponse(w, http.StatusCreated, saved, body.Data)
}
