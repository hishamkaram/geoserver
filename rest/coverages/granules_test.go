package coverages_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
)

const granulesEndpoint = "/rest/workspaces/topp/coveragestores/mosaic/coverages/mosaic/index/granules"
const indexEndpoint = "/rest/workspaces/topp/coveragestores/mosaic/coverages/mosaic/index"

func granulesClient(t *testing.T, srv *httptest.Server) *coverages.GranulesClient {
	t.Helper()
	c := newTestClient(t, srv)
	return c.Coverages.InWorkspace("topp").InCoverageStore("mosaic").Granules("mosaic")
}

func TestGranules_Schema_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != indexEndpoint {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Schema":{"attributes":{"Attribute":[
			{"name":"the_geom","minOccurs":0,"maxOccurs":1,"nillable":true,"binding":"org.locationtech.jts.geom.MultiPolygon"},
			{"name":"location","minOccurs":0,"maxOccurs":1,"nillable":true,"binding":"java.lang.String","length":254}
		]},"href":"http://srv/rest/workspaces/topp/coveragestores/mosaic/coverages/mosaic/index/granules.json"}}`)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	schema, err := g.Schema(context.Background())
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if len(schema.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(schema.Attributes))
	}
	if schema.Attributes[1].Name != "location" || schema.Attributes[1].Length != 254 {
		t.Errorf("attr[1] = %+v", schema.Attributes[1])
	}
	if !strings.Contains(schema.Href, "/granules.json") {
		t.Errorf("Href = %q", schema.Href)
	}
}

func TestGranules_List_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != granulesEndpoint {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		// No filter / offset / limit when ListGranulesOptions is zero-valued.
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"type":"FeatureCollection","features":[
			{"type":"Feature","id":"mosaic.1","geometry":{"type":"Point","coordinates":[0,0]},"properties":{"location":"a.png"}},
			{"type":"Feature","id":"mosaic.2","geometry":{"type":"Point","coordinates":[1,1]},"properties":{"location":"b.png"}}
		]}`)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	list, err := g.List(context.Background(), coverages.ListGranulesOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d", len(list))
	}
	if list[0].ID != "mosaic.1" || list[0].Properties["location"] != "a.png" {
		t.Errorf("granule[0] = %+v", list[0])
	}
}

func TestGranules_List_PassesQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter"); got != "location LIKE '%.png'" {
			t.Errorf("filter = %q", got)
		}
		if got := r.URL.Query().Get("offset"); got != "10" {
			t.Errorf("offset = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"features":[]}`)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	_, err := g.List(context.Background(), coverages.ListGranulesOptions{
		Filter: "location LIKE '%.png'",
		Offset: 10,
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestGranules_Get_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != granulesEndpoint+"/mosaic.1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"features":[{"type":"Feature","id":"mosaic.1","geometry":{"type":"Point","coordinates":[0,0]},"properties":{"location":"a.png"}}]}`)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	gr, err := g.Get(context.Background(), "mosaic.1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gr == nil || gr.ID != "mosaic.1" {
		t.Errorf("granule = %+v", gr)
	}
}

func TestGranules_Get_EmptyCollectionReturnsNil(t *testing.T) {
	// Some GeoServer 2.x versions return an empty FeatureCollection
	// instead of 404 when a granule ID isn't in the index. The SDK
	// surfaces that as (nil, nil).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"features":[]}`)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	gr, err := g.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gr != nil {
		t.Errorf("expected nil granule, got %+v", gr)
	}
}

func TestGranules_Get_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	_, err := g.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGranules_Delete_OneOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != granulesEndpoint+"/mosaic.1" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("purge") != "metadata" {
			t.Errorf("purge = %q", r.URL.Query().Get("purge"))
		}
		if r.URL.Query().Get("updateBBox") != "true" {
			t.Errorf("updateBBox = %q", r.URL.Query().Get("updateBBox"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	err := g.Delete(context.Background(), "mosaic.1", coverages.DeleteGranuleOptions{
		Purge:      coverages.PurgeMetadata,
		UpdateBBox: true,
	})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGranules_DeleteByFilter_RequiresFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	err := g.DeleteByFilter(context.Background(), coverages.DeleteGranulesOptions{})
	if err == nil || !strings.Contains(err.Error(), "INCLUDE") {
		t.Fatalf("expected refusal-without-filter error, got %v", err)
	}
}

func TestGranules_DeleteByFilter_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != granulesEndpoint {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("filter"); got != "INCLUDE" {
			t.Errorf("filter = %q", got)
		}
		if got := r.URL.Query().Get("purge"); got != "all" {
			t.Errorf("purge = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := granulesClient(t, srv)
	err := g.DeleteByFilter(context.Background(), coverages.DeleteGranulesOptions{
		Filter: "INCLUDE",
		Purge:  coverages.PurgeAll,
	})
	if err != nil {
		t.Fatalf("DeleteByFilter: %v", err)
	}
}

func TestGranules_AccessorsExposeScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()
	c := newTestClient(t, srv)
	g := c.Coverages.InWorkspace("nurc").InCoverageStore("mosaic").Granules("mosaic")
	if g.Workspace() != "nurc" || g.CoverageStore() != "mosaic" || g.Coverage() != "mosaic" {
		t.Errorf("scope accessors = %q/%q/%q", g.Workspace(), g.CoverageStore(), g.Coverage())
	}
}
