package geoserver

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestCatalog returns a *GeoServer pointing at the given httptest.Server,
// pre-configured with basic-auth and a discard logger.
func newTestCatalog(srv *httptest.Server) *GeoServer {
	return New(srv.URL+"/", "admin", "geoserver", WithLogger(nil))
}

func TestWorkspaces_GetWorkspaces_OK(t *testing.T) {
	want := `{"workspaces":{"workspace":[{"name":"topp","href":"http://localhost/rest/workspaces/topp.json"}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/workspaces", r.URL.Path)
		assert.Equal(t, "Basic "+basicAuth("admin", "geoserver"), r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, want)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	got, err := gs.GetWorkspacesContext(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "topp", got[0].Name)
}

func TestWorkspaces_GetWorkspaces_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "missing")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetWorkspacesContext(context.Background())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWorkspaces_CreateWorkspace_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/workspaces", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"name":"new_ws"`)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateWorkspaceContext(context.Background(), "new_ws")
	assert.NoError(t, err)
	assert.True(t, created)
}

func TestWorkspaces_CreateWorkspace_409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "Workspace already exists")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateWorkspaceContext(context.Background(), "existing")
	assert.False(t, created)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestWorkspaces_DeleteWorkspace_recurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/rest/workspaces/topp", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("recurse"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteWorkspaceContext(context.Background(), "topp", true)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

// TestWorkspaces_URLEscaping verifies the v1.1 PathEscape fix: workspace
// names with spaces / non-ASCII chars produce correctly-escaped URLs rather
// than malformed ones, AND the encoding is single (not double) — see the
// ParseURL RawPath fix.
func TestWorkspaces_URLEscaping(t *testing.T) {
	var capturedRequestURI, capturedEscaped string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURI = r.RequestURI     // raw bytes from the wire
		capturedEscaped = r.URL.EscapedPath() // canonical encoded form
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"workspace":{"name":"my workspace"}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetWorkspaceContext(context.Background(), "my workspace")
	assert.NoError(t, err)
	if !strings.Contains(capturedRequestURI, "my%20workspace") {
		t.Fatalf("expected wire URI to contain percent-encoded space, got %q", capturedRequestURI)
	}
	if !strings.Contains(capturedEscaped, "my%20workspace") {
		t.Fatalf("expected EscapedPath to contain percent-encoded space, got %q", capturedEscaped)
	}
	// Regression guard: must NOT be double-encoded (would yield "%2520").
	if strings.Contains(capturedRequestURI, "%2520") {
		t.Fatalf("URL is double-encoded: %q", capturedRequestURI)
	}
}

// basicAuth replicates the Authorization header value http.Request.SetBasicAuth
// produces, for assertion in handlers.
func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}
