package files

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GoLessons/sufir-keeper-server/internal/testutil"
)

func TestSanitizeHeaderFilenameVariants(t *testing.T) {
	if sanitizeHeaderFilename("") != "" {
		t.Fatalf("expected empty")
	}
	if sanitizeHeaderFilename(" a/b/c ") != "c" {
		t.Fatalf("expected base name")
	}
	if sanitizeHeaderFilename(" \n  a\t ") != "a" {
		t.Fatalf("expected trimmed name")
	}
	long := make([]byte, 300)
	for i := 0; i < len(long); i++ {
		long[i] = 'x'
	}
	if len(sanitizeHeaderFilename(string(long))) != 255 {
		t.Fatalf("expected 255 length")
	}
}

func TestUserIDFromRequest(t *testing.T) {
	userID := uuid.New()
	body := map[string]string{"a": "b"}
	data, _ := json.Marshal(body)
	req := testutil.AuthorizedJSONRequest(http.MethodGet, "/items", userID, data)
	id, ok := UserIDFromRequest(req)
	require.True(t, ok)
	require.Equal(t, userID, id)
}
