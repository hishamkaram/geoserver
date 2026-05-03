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

func TestFeatureTypes_CreateFeatureType_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/workspaces/topp/datastores/states_pg/featuretypes", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"featureType"`)
		assert.Contains(t, string(body), `"name":"polygon"`)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	ft := &FeatureType{Name: "polygon", Title: "polygon", Srs: "EPSG:4326"}
	created, err := gs.CreateFeatureTypeContext(context.Background(), "topp", "states_pg", ft)
	assert.NoError(t, err)
	assert.True(t, created)
}

func TestFeatureTypes_CreateFeatureType_400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "no attributes")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateFeatureTypeContext(context.Background(), "topp", "states_pg", &FeatureType{Name: "polygon"})
	assert.False(t, created)
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestFeatureTypes_CreateFeatureType_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "datastore not found")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateFeatureTypeContext(context.Background(), "missing-ws", "missing-ds", &FeatureType{Name: "x"})
	assert.False(t, created)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFeatureTypes_FeatureTypeServiceImpl(t *testing.T) {
	var _ FeatureTypeService = (*GeoServer)(nil)
	var _ FeatureTypeServiceWithContext = (*GeoServer)(nil)
}

func TestFeatureTypes_GetFeatureTypeList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/workspaces/topp/datastores/states_pg/featuretypes", r.URL.Path)
		assert.Equal(t, "available", r.URL.Query().Get("list"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":["counties","cities","states"]}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	got, err := gs.GetFeatureTypeListContext(context.Background(), "topp", "states_pg", FeatureTypeListAvailable)
	assert.NoError(t, err)
	assert.Equal(t, []string{"counties", "cities", "states"}, got)
}

func TestFeatureTypes_GetFeatureTypeList_DefaultKind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty kind must default to "all".
		assert.Equal(t, "all", r.URL.Query().Get("list"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":[]}}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	got, err := gs.GetFeatureTypeListContext(context.Background(), "topp", "states_pg", "")
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestFeatureTypes_GetFeatureTypeList_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "no such datastore")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetFeatureTypeListContext(context.Background(), "topp", "missing_ds", FeatureTypeListAll)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
