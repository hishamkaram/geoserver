package geoserver

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorIsSentinel(t *testing.T) {
	cases := []struct {
		name         string
		status       int
		wantSentinel error
	}{
		{"400 BadRequest", http.StatusBadRequest, ErrBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized, ErrUnauthorized},
		{"403 Forbidden", http.StatusForbidden, ErrForbidden},
		{"404 NotFound", http.StatusNotFound, ErrNotFound},
		{"405 MethodNotAllowed", http.StatusMethodNotAllowed, ErrMethodNotAllowed},
		{"409 Conflict", http.StatusConflict, ErrConflict},
		{"415 UnsupportedMediaType", http.StatusUnsupportedMediaType, ErrUnsupportedMediaType},
		{"429 TooManyRequests", http.StatusTooManyRequests, ErrRateLimited},
		{"500 InternalServerError", http.StatusInternalServerError, ErrServerError},
		{"502 BadGateway", http.StatusBadGateway, ErrServerError},
		{"503 ServiceUnavailable", http.StatusServiceUnavailable, ErrServerError},
		{"504 GatewayTimeout", http.StatusGatewayTimeout, ErrServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := newError("Op", "/rest/whatever", tc.status, []byte("body"))
			assert.True(t, errors.Is(err, tc.wantSentinel),
				"errors.Is(err, %v) should be true for status %d", tc.wantSentinel, tc.status)
		})
	}
}

func TestErrorAsTyped(t *testing.T) {
	err := newError("GetWorkspace", "/rest/workspaces/topp", 404, []byte("nope"))
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should retrieve *Error")
	}
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "GetWorkspace", apiErr.Op)
	assert.Equal(t, "/rest/workspaces/topp", apiErr.URL)
	assert.Equal(t, []byte("nope"), apiErr.Body)
}

// TestErrorBodyTruncation verifies that very large response bodies are
// capped so log output and error strings stay reasonable.
func TestErrorBodyTruncation(t *testing.T) {
	huge := make([]byte, maxBodyBytes*2)
	for i := range huge {
		huge[i] = 'x'
	}
	err := newError("X", "/x", 500, huge)
	assert.Equal(t, maxBodyBytes, len(err.Body))
}

// TestGetErrorPreservesHistoricFormat asserts that the v1.0
// "abstract:%s\ndetails:%s\n" string format is unchanged after the v1.1
// migration to *Error. Callers that pattern-match on this text continue to
// work.
func TestGetErrorPreservesHistoricFormat(t *testing.T) {
	g := New("http://example.invalid/", "u", "p")
	err := g.GetError(404, []byte("Custom Error"))

	want := "abstract:Not Found\ndetails:Custom Error\n"
	assert.Equal(t, want, err.Error())
	assert.True(t, errors.Is(err, ErrNotFound))
}

// TestGetErrorUnknownStatus verifies the "unexpected error with status code"
// fallback for a status not present in statusErrorMapping.
func TestGetErrorUnknownStatus(t *testing.T) {
	g := New("http://example.invalid/", "u", "p")
	err := g.GetError(418, []byte("teapot"))
	assert.True(t, strings.HasPrefix(err.Error(), "abstract:unexpected error with status code 418\n"))
}
