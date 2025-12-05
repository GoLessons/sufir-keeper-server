package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type LogoutHandler struct {
	users     *repository.UserRepository
	tokenAuth *jwtauth.JWTAuth
}

func NewLogoutHandler(users *repository.UserRepository, tokenAuth *jwtauth.JWTAuth) *LogoutHandler {
	return &LogoutHandler{users: users, tokenAuth: tokenAuth}
}

func (h *LogoutHandler) Handle(w http.ResponseWriter, r *http.Request) {
	tok, claims, err := jwtauth.FromContext(r.Context())
	if err != nil || tok == nil || claims == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}
	sub, _ := claims["sub"].(string)
	if strings.TrimSpace(sub) == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid subject")
		return
	}
	userID, err := uuid.Parse(sub)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid subject")
		return
	}
	if _, err := h.users.IncrementRefreshVersion(r.Context(), userID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
