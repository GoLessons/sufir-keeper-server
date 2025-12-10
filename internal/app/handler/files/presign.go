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
	Size     int64     `json:"size"`
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
	if req.FileID == uuid.Nil || strings.TrimSpace(req.Filename) == "" || strings.TrimSpace(req.Mime) == "" || req.Size <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Missing required fields")
		return
	}
	meta := map[string]string{
		"x-amz-meta-user-id":  userID.String(),
		"x-amz-meta-file-id":  req.FileID.String(),
		"x-amz-meta-filename": strings.TrimSpace(req.Filename),
		"x-amz-meta-mime":     strings.TrimSpace(req.Mime),
	}
	if s := strings.TrimSpace(req.Checksum); s != "" {
		meta["x-amz-meta-checksum"] = s
	}
	_, fields, err := h.s3.PresignPost(r.Context(), req.FileID.String(), req.Mime, req.Size, meta, time.Hour)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Presign error")
		return
	}
	fields["success_action_status"] = "204"
	resp := presignResponse{UploadURL: "/api/v1/files", Key: req.FileID.String(), FormFields: fields}
	httputil.WriteJSON(w, http.StatusOK, resp)
}
