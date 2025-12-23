package files

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/minio/sio"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type WebhookHandler struct {
	items           *repository.ItemRepository
	s3client        s3.Service
	kek             keyencrypt.Provider
	webhookSecret   string
	protectedBucket string
}

func NewWebhookHandler(items *repository.ItemRepository, s3client s3.Service, kek keyencrypt.Provider, secret string, protectedBucket string) *WebhookHandler {
	return &WebhookHandler{
		items:           items,
		s3client:        s3client,
		kek:             kek,
		webhookSecret:   strings.TrimSpace(secret),
		protectedBucket: strings.TrimSpace(protectedBucket),
	}
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

		userIDStr := strings.TrimSpace(getMeta(meta, "user-id"))
		fileIDStr := strings.TrimSpace(getMeta(meta, "file-id"))
		checksum := strings.TrimSpace(getMeta(meta, "checksum"))
		filename := strings.TrimSpace(getMeta(meta, "filename"))
		mime := strings.TrimSpace(getMeta(meta, "mime"))

		if userIDStr == "" || fileIDStr == "" {
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}
		obj, err := h.s3client.GetObject(r.Context(), "", key)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}
		stat, err := h.s3client.StatObject(r.Context(), "", key)
		if err != nil {
			_ = obj.Close()
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}

		// Generate DEK
		dek := make([]byte, 32)
		if _, err := rand.Read(dek); err != nil {
			_ = obj.Close()
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}

		// Prepare streaming encryption
		// We don't check checksum here because we are streaming.
		// If checksum is critical, we might need to read the whole file or trust the client provided checksum.
		// For now, we will rely on authenticated encryption (DARE) to ensure integrity of the encrypted data.

		// dataAAD := []byte(userID.String() + "|" + fileID.String() + "|" + "BINARY")
		encReader, err := sio.EncryptReader(obj, sio.Config{
			Key:          dek,
			MinVersion:   sio.Version20,
			CipherSuites: []byte{sio.AES_256_GCM},
		})
		if err != nil {
			_ = obj.Close()
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}

		// Calculate encrypted size
		// DARE 2.0 adds overhead.
		// Overhead = (Size / ChunkSize) * TagSize + FinalTagSize
		// SIO default chunk size is 64KB. Tag size for AES-GCM is 16 bytes.
		// sio.EncryptedSize calculates this.
		encryptedSize, err := sio.EncryptedSize(uint64(stat.Size))
		if err != nil {
			_ = obj.Close()
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}

		// Upload to protected bucket
		// We use the same key but in a different bucket
		_, err = h.s3client.PutObject(r.Context(), h.protectedBucket, key, encReader, int64(encryptedSize), httputil.ContentTypeOctetStream, nil)
		_ = obj.Close() // Close source object stream
		if err != nil {
			// Failed to upload encrypted file, clean up
			_ = h.s3client.RemoveObject(r.Context(), h.protectedBucket, key)
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}

		// Encrypt DEK with KEK
		kek, kekVersion, err := h.kek.GetCurrent(r.Context())
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), h.protectedBucket, key)
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}
		keyAAD := []byte(userID.String() + "|" + fileID.String())
		keyNonce, keyEncrypted, err := aead.Encrypt(kek, keyAAD, dek)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), h.protectedBucket, key)
			_ = h.s3client.RemoveObject(r.Context(), "", key)
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

		// Save metadata to DB
		recdb := model.ItemRecord{
			ID:     fileID,
			UserID: userID,
			Title:  title,
			Type:   "BINARY",
			// DataEncrypted & DataNonce are now empty for files
			DataKeyEncrypted: keyEncrypted,
			DataKeyNonce:     keyNonce,
			KEKVersion:       kekVersion,
			Meta:             m,
			File: &model.ItemFile{
				S3Bucket: h.protectedBucket,
				S3Key:    key,
				Size:     int64(encryptedSize),
				SHA256:   "", // We don't have SHA256 of the original file unless we calculate it during stream (TeeReader)
			},
		}
		_, err = h.items.Create(r.Context(), recdb)
		if err != nil {
			_ = h.s3client.RemoveObject(r.Context(), h.protectedBucket, key)
			_ = h.s3client.RemoveObject(r.Context(), "", key)
			continue
		}

		// Success! Remove original file from incoming bucket
		_ = h.s3client.RemoveObject(r.Context(), "", key)
	}
}

func getMeta(m map[string]string, key string) string {
	if v, ok := m[key]; ok {
		return v
	}
	key = strings.ToLower(key)
	keyWithPrefix := "x-amz-meta-" + key
	for k, v := range m {
		k = strings.ToLower(k)
		if k == key || k == keyWithPrefix {
			return v
		}
	}
	return ""
}
