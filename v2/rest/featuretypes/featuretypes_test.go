package featuretypes_test

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
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
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

func expectUserAgent(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("User-Agent"); got != "geoserver-go/v2" {
		t.Fatalf("User-Agent = %q, want %q", got, "geoserver-go/v2")
	}
}

// ---- CRS marshal/unmarshal round-trip ----

func TestCRS_UnmarshalObject(t *testing.T) {
	var c featuretypes.CRS
	if err := json.Unmarshal([]byte(`{"@class":"projected","$":"EPSG:4326"}`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Class != "projected" || c.Value != "EPSG:4326" {
		t.Fatalf("CRS = %+v", c)
	}
}

func TestCRS_UnmarshalBareString(t *testing.T) {
	var c featuretypes.CRS
	if err := json.Unmarshal([]byte(`"EPSG:4326"`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Class != "string" || c.Value != "EPSG:4326" {
		t.Fatalf("CRS = %+v", c)
	}
}

func TestCRS_MarshalBareString(t *testing.T) {
	c := featuretypes.CRS{Class: "string", Value: "EPSG:4326"}
	got, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `"EPSG:4326"` {
		t.Fatalf("marshal = %s", got)
	}
}

func TestCRS_MarshalObject(t *testing.T) {
	c := featuretypes.CRS{Class: "projected", Value: "EPSG:4326"}
	got, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `{"@class":"projected","$":"EPSG:4326"}` {
		t.Fatalf("marshal = %s", got)
	}
}

func TestCRS_MarshalEmpty(t *testing.T) {
	var c featuretypes.CRS
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
		expectUserAgent(t, r)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/rest/workspaces/topp/datastores/states_pg/featuretypes" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("list") != "" {
			t.Errorf("List should not send ?list=, got %q", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"featureTypes":{"featureType":[{"name":"states"},{"name":"counties"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg").
		List(context.Background(), featuretypes.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "states" || got[1].Name != "counties" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestList_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.FeatureTypes.InWorkspace("topp").InDatastore("missing").
		List(context.Background(), featuretypes.ListOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	var apiErr *geoserver.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *geoserver.APIError, got %T", err)
	}
	if apiErr.Op != "FeatureTypes.List" {
		t.Fatalf("Op = %q", apiErr.Op)
	}
}

func TestList_EmptyWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.FeatureTypes.InWorkspace("").InDatastore("ds").
		List(context.Background(), featuretypes.ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "empty workspace") {
		t.Fatalf("expected empty-workspace error, got %v", err)
	}
}

func TestList_EmptyDatastore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.FeatureTypes.InWorkspace("topp").InDatastore("").
		List(context.Background(), featuretypes.ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "empty datastore") {
		t.Fatalf("expected empty-datastore error, got %v", err)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"featureTypes":{"featureType":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for ft, err := range c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Iter(context.Background(), featuretypes.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, ft.Name)
	}
	if len(names) != 3 || names[0] != "a" || names[2] != "c" {
		t.Fatalf("iterator yielded %v", names)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/datastores/states_pg/featuretypes/states" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"featureType":{
			"name":"states","nativeName":"states","title":"States",
			"srs":"EPSG:4326","enabled":true,
			"nativeCRS":{"@class":"projected","$":"EPSG:4326"},
			"latLonBoundingBox":{"minx":-180,"maxx":180,"miny":-90,"maxy":90,"crs":"EPSG:4326"},
			"attributes":{"attribute":[{"name":"the_geom","binding":"org.locationtech.jts.geom.MultiPolygon","nillable":true}]}
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ft, err := c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg").
		Get(context.Background(), "states")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft.Name != "states" || ft.Title != "States" || ft.SRS != "EPSG:4326" || !ft.Enabled {
		t.Fatalf("FeatureType = %+v", ft)
	}
	if ft.NativeCRS == nil || ft.NativeCRS.Class != "projected" || ft.NativeCRS.Value != "EPSG:4326" {
		t.Fatalf("NativeCRS = %+v", ft.NativeCRS)
	}
	// Bare-string CRS shape on the bounding box.
	if ft.LatLonBoundingBox == nil || ft.LatLonBoundingBox.CRS == nil ||
		ft.LatLonBoundingBox.CRS.Class != "string" || ft.LatLonBoundingBox.CRS.Value != "EPSG:4326" {
		t.Fatalf("LatLonBoundingBox.CRS = %+v", ft.LatLonBoundingBox)
	}
	if ft.LatLonBoundingBox.MinX != -180 || ft.LatLonBoundingBox.MaxY != 90 {
		t.Fatalf("LatLonBoundingBox = %+v", ft.LatLonBoundingBox.BoundingBox)
	}
	if ft.Attributes == nil || len(ft.Attributes.Attribute) != 1 ||
		ft.Attributes.Attribute[0].Name != "the_geom" {
		t.Fatalf("Attributes = %+v", ft.Attributes)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_EmptyName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Get(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Fatalf("expected empty-name error, got %v", err)
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/rest/workspaces/topp/datastores/states_pg/featuretypes" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		// Must wrap in {"featureType": {...}}; must include name and srs.
		for _, sub := range []string{
			`"featureType":`,
			`"name":"states"`,
			`"nativeName":"states"`,
			`"srs":"EPSG:4326"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg").
		Create(context.Background(), &featuretypes.FeatureType{
			Name:       "states",
			NativeName: "states",
			SRS:        "EPSG:4326",
			Enabled:    true,
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "exists")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Create(context.Background(), &featuretypes.FeatureType{Name: "dup"})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_NilFeatureType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil feature type") {
		t.Fatalf("expected nil-feature-type error, got %v", err)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Create(context.Background(), &featuretypes.FeatureType{})
	if err == nil || !strings.Contains(err.Error(), "empty Name") {
		t.Fatalf("expected empty-name error, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut ||
			r.URL.Path != "/rest/workspaces/topp/datastores/ds/featuretypes/states" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"title":"New title"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Update(context.Background(), "states", &featuretypes.FeatureType{Title: "New title"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NilFeatureType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Update(context.Background(), "states", nil)
	if err == nil || !strings.Contains(err.Error(), "nil feature type") {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete ||
			r.URL.Path != "/rest/workspaces/topp/datastores/ds/featuretypes/states" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Delete(context.Background(), "states", featuretypes.DeleteOptions{Recurse: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Delete(context.Background(), "states", featuretypes.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

// ---- Discover ----

func TestDiscover_DefaultsToAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list") != "available" {
			t.Errorf("list = %q, want available", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":["new_table_a","new_table_b"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Discover(context.Background(), featuretypes.DiscoverOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "new_table_a" {
		t.Fatalf("Discover = %v", got)
	}
}

func TestDiscover_AvailableWithGeometry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list") != "available_with_geom" {
			t.Errorf("list = %q", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":["geom_table"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Discover(context.Background(), featuretypes.DiscoverOptions{
			Kind: featuretypes.DiscoverAvailableWithGeometry,
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "geom_table" {
		t.Fatalf("Discover = %v", got)
	}
}

func TestDiscover_All(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list") != "all" {
			t.Errorf("list = %q", r.URL.Query().Get("list"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"list":{"string":["a","b","c"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.FeatureTypes.InWorkspace("topp").InDatastore("ds").
		Discover(context.Background(), featuretypes.DiscoverOptions{Kind: featuretypes.DiscoverAll})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Discover = %v", got)
	}
}

// URL escaping must single-encode every segment in the 3-deep path.
func TestGet_URLEscaping_AllSegments(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"featureType":{"name":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.FeatureTypes.InWorkspace("ws*1").InDatastore("ds*2").
		Get(context.Background(), "ft*3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"ws%2A1", "ds%2A2", "ft%2A3"} {
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
	ws := c.FeatureTypes.InWorkspace("topp")
	if ws.Workspace() != "topp" {
		t.Fatalf("Workspace() = %q", ws.Workspace())
	}
	ds := ws.InDatastore("states_pg")
	if ds.Workspace() != "topp" || ds.Datastore() != "states_pg" {
		t.Fatalf("scope = %q/%q", ds.Workspace(), ds.Datastore())
	}
}
