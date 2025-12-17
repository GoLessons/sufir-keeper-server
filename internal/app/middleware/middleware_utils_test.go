package middleware

import "testing"

func TestIsTextAndBinaryContentType(t *testing.T) {
	casesText := []string{"application/json", "text/plain; charset=utf-8", "application/x-www-form-urlencoded"}
	for _, ct := range casesText {
		if !isTextContentType(ct) {
			t.Fatalf("expected text content-type: %s", ct)
		}
	}
	casesBin := []string{"application/octet-stream", "image/png", "application/pdf"}
	for _, ct := range casesBin {
		if !isBinaryContentType(ct) {
			t.Fatalf("expected binary content-type: %s", ct)
		}
	}
}
