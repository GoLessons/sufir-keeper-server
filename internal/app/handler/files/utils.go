package files

import (
	"net/http"
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
