package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type RefreshHandler struct {
	users                  *repository.UserRepository
	tokenAuth              *jwtauth.JWTAuth
	accessTokenTTLSeconds  int
	refreshTokenTTLSeconds int
}

func NewRefreshHandler(users *repository.UserRepository, tokenAuth *jwtauth.JWTAuth, accessTokenTTLSeconds int, refreshTokenTTLSeconds int) *RefreshHandler {
	return &RefreshHandler{users: users, tokenAuth: tokenAuth, accessTokenTTLSeconds: accessTokenTTLSeconds, refreshTokenTTLSeconds: refreshTokenTTLSeconds}
}

func (h *RefreshHandler) Handle(w http.ResponseWriter, r *http.Request) {
	type refreshInput struct {
		RefreshToken string `json:"refresh_token"`
	}
	var body refreshInput
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	token, err := jwtauth.VerifyToken(h.tokenAuth, body.RefreshToken)
	if err != nil || token == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired refresh token")
		return
	}
	t, _ := token.Get("typ")
	if strings.ToLower(strings.TrimSpace(castString(t))) != "refresh" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Refresh token required")
		return
	}
	if expVal, ok := token.Get("exp"); ok {
		var expSec int64
		switch v := expVal.(type) {
		case int64:
			expSec = v
		case int:
			expSec = int64(v)
		case float64:
			expSec = int64(v)
		case float32:
			expSec = int64(v)
		}
		if expSec > 0 && time.Now().Unix() > expSec {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Token expired")
			return
		}
	}
	var subject string
	var version int
	if sub, ok := token.Get("sub"); ok {
		subject = strings.TrimSpace(castString(sub))
	}
	if ver, ok := token.Get("ver"); ok {
		switch v := ver.(type) {
		case int:
			version = v
		case int32:
			version = int(v)
		case int64:
			version = int(v)
		case float64:
			version = int(v)
		case float32:
			version = int(v)
		default:
			if vstr := strings.TrimSpace(castString(ver)); vstr != "" {
				for i := 0; i < len(vstr); i++ {
					if vstr[i] < '0' || vstr[i] > '9' {
						vstr = ""
						break
					}
				}
				if vstr != "" {
					for _, ch := range vstr {
						version = version*10 + int(ch-'0')
					}
				}
			}
		}
	}
	if strings.TrimSpace(subject) == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid token subject")
		return
	}
	userID, err := uuid.Parse(subject)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid subject")
		return
	}
	currentVersion, err := h.users.GetRefreshVersion(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired refresh token")
		return
	}
	if version != currentVersion {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Refresh token revoked")
		return
	}
	expiresAccess := time.Now().Add(time.Duration(h.accessTokenTTLSeconds) * time.Second)
	_, accessToken, _ := h.tokenAuth.Encode(map[string]interface{}{"sub": subject, "exp": expiresAccess.Unix(), "typ": "access"})
	expiresRefresh := time.Now().Add(time.Duration(h.refreshTokenTTLSeconds) * time.Second)
	newVersion, err := h.users.IncrementRefreshVersion(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	_, refreshToken, _ := h.tokenAuth.Encode(map[string]interface{}{"sub": subject, "exp": expiresRefresh.Unix(), "typ": "refresh", "ver": newVersion})
	tokenType := "bearer"
	expiresIn := h.accessTokenTTLSeconds
	type authResponseOutput struct {
		AccessToken  *string `json:"access_token,omitempty"`
		ExpiresIn    *int    `json:"expires_in,omitempty"`
		RefreshToken *string `json:"refresh_token,omitempty"`
		TokenType    *string `json:"token_type,omitempty"`
	}
	response := authResponseOutput{AccessToken: &accessToken, RefreshToken: &refreshToken, TokenType: &tokenType, ExpiresIn: &expiresIn}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func castString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
