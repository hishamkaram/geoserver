package geoserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStyles_GetStyles_workspaceScoped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/workspaces/topp/styles", r.URL.Path)
		_, _ = io.WriteString(w, `{"styles":{"style":[{"name":"airports","href":"x"}]}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	got, err := gs.GetStylesContext(context.Background(), "topp")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "airports", got[0].Name)
}

func TestStyles_GetStyles_global(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = io.WriteString(w, `{"styles":{"style":[]}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetStylesContext(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, "/rest/styles", capturedPath, "empty workspace must hit the global endpoint, not /rest/workspaces//styles")
}

func TestStyles_CreateStyle_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateStyleContext(context.Background(), "topp", "x")
	assert.False(t, created)
	if !errors.Is(err, ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

func TestStyles_DeleteStyle_purgeQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/workspaces/topp/styles/x", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("purge"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteStyleContext(context.Background(), "topp", "x", true)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestStyles_StyleExists_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	exists, err := gs.StyleExistsContext(context.Background(), "topp", "missing")
	assert.False(t, exists)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStyles_UploadStyle_passesSLDContentType(t *testing.T) {
	step := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch step {
		case 0:
			// UploadStyle calls StyleExists first; return 404 so we proceed.
			step = 1
			w.WriteHeader(http.StatusNotFound)
		case 1:
			// CreateStyle.
			assert.Equal(t, http.MethodPost, r.Method)
			step = 2
			w.WriteHeader(http.StatusCreated)
		case 2:
			// Actual SLD PUT — the only step that should carry SLD content-type.
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, "application/vnd.ogc.sld+xml", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	ok, err := gs.UploadStyleContext(context.Background(), strings.NewReader(`<sld/>`), "topp", "x", false)
	assert.NoError(t, err)
	assert.True(t, ok)
}
