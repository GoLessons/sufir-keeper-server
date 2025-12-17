package testutil

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func AuthorizedJSONRequest(method string, path string, userID uuid.UUID, jsonBody []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	token := jwt.New()
	_ = token.Set("sub", userID.String())
	_ = token.Set("typ", "access")
	return req.WithContext(jwtauth.NewContext(req.Context(), token, nil))
}
