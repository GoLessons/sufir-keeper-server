package auth

import (
	"net/http"

	fileshandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/files"
)

type VerifyHandler struct{}

func NewVerifyHandler() *VerifyHandler { return &VerifyHandler{} }

func (h *VerifyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, exists := fileshandler.UserIDFromRequest(r)
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("X-User-Id", userID.String())
	w.WriteHeader(http.StatusNoContent)
}
