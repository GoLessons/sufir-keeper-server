package files

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type PresignHandler struct {
	s3 *s3.Client
}

func NewPresignHandler(s3c *s3.Client) *PresignHandler {
	return &PresignHandler{s3: s3c}
}

type presignRequest struct {
	Filename string    `json:"filename"`
	Mime     string    `json:"mime"`
	Checksum string    `json:"checksum"`
	FileID   uuid.UUID `json:"fileId"`
}

type presignResponse struct {
	FormFields map[string]string `json:"form_fields"`
	UploadURL  string            `json:"upload_url"`
	Key        string            `json:"key"`
}

func (h *PresignHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromRequest(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	var req presignRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid JSON")
		return
	}
	if req.FileID == uuid.Nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Missing required fields")
		return
	}
	meta := map[string]string{
		"user-id": userID.String(),
		"file-id": req.FileID.String(),
	}
	if s := strings.TrimSpace(req.Filename); s != "" {
		meta["filename"] = s
	}
	if s := strings.TrimSpace(req.Mime); s != "" {
		meta["mime"] = s
	}
	if s := strings.TrimSpace(req.Checksum); s != "" {
		meta["checksum"] = s
	}
	uploadURL, fields, err := h.s3.PresignPost(r.Context(), req.FileID.String(), req.Mime, 0, meta, time.Hour)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Presign error")
		return
	}
	fields["success_action_status"] = "204"
	_ = uploadURL
	resp := presignResponse{UploadURL: "/api/v1/files", Key: req.FileID.String(), FormFields: fields}
	httputil.WriteJSON(w, http.StatusOK, resp)
}
