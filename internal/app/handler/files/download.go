package files

import (
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/sio"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/aead"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type DownloadHandler struct {
	itemsRepo *repository.ItemRepository
	s3client  *s3.Client
	kek       keyencrypt.Provider
}

func NewDownloadHandler(items *repository.ItemRepository, s3client *s3.Client, kek keyencrypt.Provider) *DownloadHandler {
	return &DownloadHandler{itemsRepo: items, s3client: s3client, kek: kek}
}

func (h *DownloadHandler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID, ok := UserIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	rec, err := h.itemsRepo.Get(r.Context(), userID, id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "File not found")
		return
	}
	if strings.ToUpper(strings.TrimSpace(rec.Type)) != "BINARY" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}
	kek, err := h.kek.GetByVersion(r.Context(), rec.KEKVersion)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "KEK error")
		return
	}
	dataKeyAAD := []byte(rec.UserID.String() + "|" + rec.ID.String())
	dek, err := aead.Decrypt(kek, dataKeyAAD, rec.DataKeyNonce, rec.DataKeyEncrypted)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Decrypt key error")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	if strings.TrimSpace(rec.Title) != "" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+rec.Title+"\"")
	} else if rec.Meta != nil {
		if v, ok := rec.Meta["filename"]; ok && strings.TrimSpace(v) != "" {
			w.Header().Set("Content-Disposition", "attachment; filename=\""+v+"\"")
		}
		if v, ok := rec.Meta["mime"]; ok && strings.TrimSpace(v) != "" {
			w.Header().Set("Content-Type", v)
		}
	}

	// Case 1: File is in S3 (New format)
	if rec.File != nil {
		obj, err := h.s3client.GetObject(r.Context(), rec.File.S3Bucket, rec.File.S3Key)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "File object not found")
			return
		}

		decReader, err := sio.DecryptReader(obj, sio.Config{
			Key:          dek,
			MinVersion:   sio.Version20,
			CipherSuites: []byte{sio.AES_256_GCM},
		})
		if err != nil {
			_ = obj.Close()
			httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Decrypt init error")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, decReader)
		_ = obj.Close()
		return
	}

	// Case 2: File is in DB (Old format)
	dataAAD := []byte(rec.UserID.String() + "|" + rec.ID.String() + "|" + rec.Type)
	bytes, err := aead.Decrypt(dek, dataAAD, rec.DataNonce, rec.DataEncrypted)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Decrypt error")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}
