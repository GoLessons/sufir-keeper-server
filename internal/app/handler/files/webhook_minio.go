package files

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type WebhookHandler struct {
	items         *repository.ItemRepository
	s3client      *s3.Client
	kek           keyencrypt.Provider
	webhookSecret string
}

func NewWebhookHandler(items *repository.ItemRepository, s3client *s3.Client, kek keyencrypt.Provider, secret string) *WebhookHandler {
	return &WebhookHandler{items: items, s3client: s3client, kek: kek, webhookSecret: strings.TrimSpace(secret)}
}

type minioEvent struct {
	EventName string `json:"eventName"`
	Records   []struct {
		S3 struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				UserMetadata map[string]string `json:"userMetadata"`
				Key          string            `json:"key"`
				ContentType  string            `json:"contentType"`
				Size         int64             `json:"size"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		auth = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if h.webhookSecret != "" && auth != h.webhookSecret {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid webhook secret")
		return
	}
	var ev minioEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	for _, rec := range ev.Records {
		key := strings.TrimSpace(rec.S3.Object.Key)
		meta := rec.S3.Object.UserMetadata
		userIDStr := strings.TrimSpace(meta["user-id"])
		fileIDStr := strings.TrimSpace(meta["file-id"])
		checksum := strings.TrimSpace(meta["checksum"])
		filename := strings.TrimSpace(meta["filename"])
		mime := strings.TrimSpace(meta["mime"])
		if userIDStr == "" || fileIDStr == "" {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		obj, err := h.s3client.GetObject(r.Context(), key)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		stat, err := obj.Stat()
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		buf := make([]byte, stat.Size)
		n, _ := io.ReadFull(obj, buf)
		_ = obj.Close()
		if int64(n) != stat.Size {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		if checksum != "" {
			s := sha256.Sum256(buf)
			if !strings.EqualFold(hex.EncodeToString(s[:]), checksum) {
				_ = h.s3client.RemoveObject(r.Context(), key)
				continue
			}
		}
		dek := make([]byte, 32)
		if _, err := rand.Read(dek); err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		dataAAD := []byte(userID.String() + "|" + fileID.String() + "|" + "BINARY")
		dataNonce, dataEncrypted, err := aead.Encrypt(dek, dataAAD, buf)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		kek, kekVersion, err := h.kek.GetCurrent(r.Context())
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		keyAAD := []byte(userID.String() + "|" + fileID.String())
		keyNonce, keyEncrypted, err := aead.Encrypt(kek, keyAAD, dek)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		m := map[string]string{"checksum": checksum}
		if filename != "" {
			m["filename"] = filename
		}
		if mime != "" {
			m["mime"] = mime
		}
		title := filename
		if strings.TrimSpace(title) == "" {
			title = fileID.String()
		}
		recdb := model.ItemRecord{ID: fileID, UserID: userID, Title: title, Type: "BINARY", DataEncrypted: dataEncrypted, DataNonce: dataNonce, DataKeyEncrypted: keyEncrypted, DataKeyNonce: keyNonce, KEKVersion: kekVersion, Meta: m}
		_, err = h.items.Create(r.Context(), recdb)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), key)
			continue
		}
		_ = h.s3client.RemoveObject(r.Context(), key)
	}
	w.WriteHeader(http.StatusNoContent)
}
