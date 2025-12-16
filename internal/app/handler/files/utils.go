package files

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
)

func UserIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil || claims == nil {
		return uuid.Nil, false
	}
	sub, _ := claims["sub"].(string)
	id, err := uuid.Parse(strings.TrimSpace(sub))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func sanitizeHeaderFilename(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer("\r", "", "\n", "", "\t", "")
	s = replacer.Replace(s)
	s = filepath.Base(s)
	if len(s) > 255 {
		s = s[:255]
	}
	s = strings.TrimSpace(s)
	return s
}
