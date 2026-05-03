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
