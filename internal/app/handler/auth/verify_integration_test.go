package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestVerifyHandlerIntegration(t *testing.T) {
	handler := NewVerifyHandler()

	userID := uuid.New()
	req := testutil.AuthorizedJSONRequest(http.MethodPost, "/auth-verify", userID, nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Equal(t, userID.String(), rr.Header().Get("X-User-Id"))
}
