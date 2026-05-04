package coverages_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
)

func newTestClient(t *testing.T, srv *httptest.Server) *geoserver.Client {
	t.Helper()
	c, err := geoserver.New(srv.URL,
		geoserver.WithBasicAuth("admin", "geoserver"),
		geoserver.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func expectBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:geoserver"))
	if got := r.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization header = %q, want %q", got, want)
	}
}

// ---- CRS marshal/unmarshal round-trip ----

func TestCRS_UnmarshalObject(t *testing.T) {
	var c coverages.CRS
	if err := json.Unmarshal([]byte(`{"@class":"projected","$":"EPSG:3857"}`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Class != "projected" || c.Value != "EPSG:3857" {
		t.Fatalf("CRS = %+v", c)
	}
}

func TestCRS_UnmarshalBareString(t *testing.T) {
	var c coverages.CRS
	if err := json.Unmarshal([]byte(`"EPSG:3857"`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Class != "string" || c.Value != "EPSG:3857" {
		t.Fatalf("CRS = %+v", c)
	}
}

func TestCRS_MarshalEmpty(t *testing.T) {
	var c coverages.CRS
	got, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `""` {
		t.Fatalf("marshal = %s", got)
	}
}

// ---- HTTP CRUD tests ----

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.URL.Path != "/rest/workspaces/ne/coveragestores/states_tiff/coverages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("list") != "" {
			t.Errorf("List should not send ?list=, got %q", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverages":{"coverage":[{"name":"states"},{"name":"counties"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Coverages.InWorkspace("ne").InCoverageStore("states_tiff").
		List(context.Background(), coverages.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "states" || got[1].Name != "counties" {
		t.Fatalf("List = %+v", got)
	}
}

func TestList_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Coverages.InWorkspace("ne").InCoverageStore("missing").
		List(context.Background(), coverages.ListOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	var apiErr *geoserver.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *geoserver.APIError, got %T", err)
	}
	if apiErr.Op != "Coverages.List" {
		t.Fatalf("Op = %q", apiErr.Op)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverages":{"coverage":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for cov, err := range c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Iter(context.Background(), coverages.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, cov.Name)
	}
	if len(names) != 3 {
		t.Fatalf("Iter = %v", names)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/ne/coveragestores/states_tiff/coverages/states" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverage":{
			"name":"states","nativeName":"states","nativeFormat":"GeoTIFF",
			"srs":"EPSG:4326","enabled":true,
			"nativeCRS":{"@class":"projected","$":"EPSG:4326"},
			"latLonBoundingBox":{"minx":-180,"maxx":180,"miny":-90,"maxy":90,"crs":"EPSG:4326"}
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	cov, err := c.Coverages.InWorkspace("ne").InCoverageStore("states_tiff").
		Get(context.Background(), "states")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cov.Name != "states" || cov.NativeFormat != "GeoTIFF" || cov.SRS != "EPSG:4326" || !cov.Enabled {
		t.Fatalf("Coverage = %+v", cov)
	}
	if cov.NativeCRS == nil || cov.NativeCRS.Class != "projected" {
		t.Fatalf("NativeCRS = %+v", cov.NativeCRS)
	}
	if cov.LatLonBoundingBox == nil || cov.LatLonBoundingBox.CRS == nil ||
		cov.LatLonBoundingBox.CRS.Class != "string" || cov.LatLonBoundingBox.CRS.Value != "EPSG:4326" {
		t.Fatalf("LatLonBoundingBox.CRS = %+v", cov.LatLonBoundingBox)
	}
	if cov.LatLonBoundingBox.MinX != -180 || cov.LatLonBoundingBox.MaxY != 90 {
		t.Fatalf("BoundingBox = %+v", cov.LatLonBoundingBox.BoundingBox)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/rest/workspaces/ne/coveragestores/states_tiff/coverages" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, sub := range []string{
			`"coverage":`,
			`"name":"states_published"`,
			`"nativeCoverageName":"states.tif"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Coverages.InWorkspace("ne").InCoverageStore("states_tiff").
		Create(context.Background(), &coverages.Coverage{
			Name:               "states_published",
			NativeCoverageName: "states.tif",
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Create(context.Background(), &coverages.Coverage{Name: "dup"})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_NilCoverage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil coverage") {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut ||
			r.URL.Path != "/rest/workspaces/ne/coveragestores/cs/coverages/states" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"title":"Updated"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Update(context.Background(), "states", &coverages.Coverage{Title: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete ||
			r.URL.Path != "/rest/workspaces/ne/coveragestores/cs/coverages/states" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Delete(context.Background(), "states", coverages.DeleteOptions{Recurse: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Delete(context.Background(), "states", coverages.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

// ---- Discover ----

func TestDiscover_DefaultsToAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list") != "all" {
			t.Errorf("list = %q, want all", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":["states","counties"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Discover(context.Background(), coverages.DiscoverOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "states" {
		t.Fatalf("Discover = %v", got)
	}
}

func TestDiscover_Available(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list") != "available" {
			t.Errorf("list = %q, want available", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":["new_raster"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Coverages.InWorkspace("ne").InCoverageStore("cs").
		Discover(context.Background(), coverages.DiscoverOptions{Kind: coverages.DiscoverAvailable})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "new_raster" {
		t.Fatalf("Discover = %v", got)
	}
}

// URL escaping must single-encode every segment in the 3-deep path.
func TestGet_URLEscaping_AllSegments(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverage":{"name":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Coverages.InWorkspace("ws*1").InCoverageStore("cs*2").
		Get(context.Background(), "cov*3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"ws%2A1", "cs%2A2", "cov%2A3"} {
		if !strings.Contains(capturedURI, want) {
			t.Fatalf("expected %q in %q", want, capturedURI)
		}
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}

func TestScopeAccessors(t *testing.T) {
	c, err := geoserver.New("http://localhost:8080/geoserver")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ws := c.Coverages.InWorkspace("ne")
	if ws.Workspace() != "ne" {
		t.Fatalf("Workspace() = %q", ws.Workspace())
	}
	cs := ws.InCoverageStore("states_tiff")
	if cs.Workspace() != "ne" || cs.CoverageStore() != "states_tiff" {
		t.Fatalf("scope = %q/%q", cs.Workspace(), cs.CoverageStore())
	}
}
