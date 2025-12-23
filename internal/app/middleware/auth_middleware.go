package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/jwtauth/v5"

	"github.com/GoLessons/sufir-keeper-server/internal/api"
	model "github.com/GoLessons/sufir-keeper-server/internal/api/types"
)

func AuthRequiredMiddleware(tokenAuth *jwtauth.JWTAuth) api.MiddlewareFunc {
	verifier := jwtauth.Verifier(tokenAuth)
	authenticator := jwtauth.Authenticator(tokenAuth)

	return func(next http.Handler) http.Handler {
		return verifier(
			authenticator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tok, claims, err := jwtauth.FromContext(r.Context())
				if err != nil || tok == nil || claims == nil {
					status := http.StatusUnauthorized
					code := "unauthorized"
					message := "Invalid or expired token"
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_ = json.NewEncoder(w).Encode(model.Error{Code: &status, Error: &code, Message: &message})
					return
				}

				t, _ := claims["typ"].(string)
				if strings.ToLower(strings.TrimSpace(t)) != "access" {
					status := http.StatusUnauthorized
					code := "unauthorized"
					message := "Access token required"
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_ = json.NewEncoder(w).Encode(model.Error{Code: &status, Error: &code, Message: &message})
					return
				}

				next.ServeHTTP(w, r)
			},
			)),
		)
	}
}
