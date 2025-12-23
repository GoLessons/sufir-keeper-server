package auth

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"

	apitypes "github.com/GoLessons/sufir-keeper-server/internal/api/types"
	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/auth"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type LoginHandler struct {
	users                  repository.UserStore
	tokenAuth              *jwtauth.JWTAuth
	accessTokenTTLSeconds  int
	refreshTokenTTLSeconds int
}

func NewLoginHandler(users repository.UserStore, tokenAuth *jwtauth.JWTAuth, accessTokenTTLSeconds int, refreshTokenTTLSeconds int) *LoginHandler {
	return &LoginHandler{users: users, tokenAuth: tokenAuth, accessTokenTTLSeconds: accessTokenTTLSeconds, refreshTokenTTLSeconds: refreshTokenTTLSeconds}
}

type loginInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var body loginInput
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	record, err := h.users.GetByLogin(r.Context(), strings.TrimSpace(body.Login))
	if err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid login or password")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	if !auth.VerifyPassword(record.PasswordHash, body.Password) {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid login or password")
		return
	}
	expiresAccess := time.Now().Add(time.Duration(h.accessTokenTTLSeconds) * time.Second)
	_, accessToken, err := h.tokenAuth.Encode(map[string]interface{}{"sub": record.ID.String(), "exp": expiresAccess.Unix(), "typ": "access"})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Token generation error")
		return
	}
	expiresRefresh := time.Now().Add(time.Duration(h.refreshTokenTTLSeconds) * time.Second)
	currentVersion, err := h.users.EnsureRefreshVersion(r.Context(), record.ID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	_, refreshToken, err := h.tokenAuth.Encode(map[string]interface{}{"sub": record.ID.String(), "exp": expiresRefresh.Unix(), "typ": "refresh", "ver": currentVersion})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Token generation error")
		return
	}
	tokenType := "bearer"
	expiresIn := h.accessTokenTTLSeconds
	response := apitypes.AuthResponse{AccessToken: &accessToken, RefreshToken: &refreshToken, TokenType: &tokenType, ExpiresIn: &expiresIn}
	httputil.WriteJSON(w, http.StatusOK, response)
}
