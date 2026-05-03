package geoserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayers_GetLayers_workspaceScoped(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = io.WriteString(w, `{"layers":{"layer":[{"name":"states","href":"x"}]}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	got, err := gs.GetLayersContext(context.Background(), "topp")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "states", got[0].Name)
	assert.Equal(t, "/rest/workspaces/topp/layers", capturedPath)
}

func TestLayers_GetLayers_globalFallback(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = io.WriteString(w, `{"layers":{"layer":[]}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetLayersContext(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, "/rest/layers", capturedPath)
}

func TestLayers_GetLayer_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetLayerContext(context.Background(), "topp", "states")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLayers_DeleteLayer_recurse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/workspaces/topp/layers/states", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("recurse"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	ok, err := gs.DeleteLayerContext(context.Background(), "topp", "states", true)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestLayers_PublishPostgisLayer_buildsCorrectURL(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	ok, err := gs.PublishPostgisLayerContext(context.Background(), "topp", "ds", "publish_name", "table_name")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "/rest/workspaces/topp/datastores/ds/featuretypes", capturedPath)
}
