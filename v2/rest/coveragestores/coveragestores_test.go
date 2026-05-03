package coveragestores_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/coveragestores"
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

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.Method != http.MethodGet || r.URL.Path != "/rest/workspaces/ne/coveragestores" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverageStores":{"coverageStore":[{"name":"states_tiff"},{"name":"world_tiff"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.CoverageStores.InWorkspace("ne").List(context.Background(), coveragestores.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "states_tiff" || got[1].Name != "world_tiff" {
		t.Fatalf("List = %+v", got)
	}
}

func TestList_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.CoverageStores.InWorkspace("missing").List(context.Background(), coveragestores.ListOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList_EmptyWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.CoverageStores.InWorkspace("").List(context.Background(), coveragestores.ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "empty workspace") {
		t.Fatalf("expected empty-workspace error, got %v", err)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverageStores":{"coverageStore":[{"name":"a"},{"name":"b"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for s, err := range c.CoverageStores.InWorkspace("ne").Iter(context.Background(), coveragestores.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, s.Name)
	}
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("Iter = %v", names)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/ne/coveragestores/states_tiff" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverageStore":{"name":"states_tiff","type":"GeoTIFF","url":"file:data/states.tif","enabled":true,"workspace":{"name":"ne"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	store, err := c.CoverageStores.InWorkspace("ne").Get(context.Background(), "states_tiff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Name != "states_tiff" || store.Type != "GeoTIFF" || !store.Enabled {
		t.Fatalf("store = %+v", store)
	}
	if store.Workspace == nil || store.Workspace.Name != "ne" {
		t.Fatalf("Workspace = %+v", store.Workspace)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.CoverageStores.InWorkspace("ne").Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/workspaces/ne/coveragestores" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, sub := range []string{
			`"coverageStore":`,
			`"name":"states_tiff"`,
			`"type":"GeoTIFF"`,
			`"url":"file:data/states.tif"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CoverageStores.InWorkspace("ne").Create(context.Background(), &coveragestores.CoverageStore{
		Name: "states_tiff", Type: "GeoTIFF", URL: "file:data/states.tif", Enabled: true,
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
	err := c.CoverageStores.InWorkspace("ne").Create(context.Background(), &coveragestores.CoverageStore{
		Name: "dup", Type: "GeoTIFF", URL: "file:data/dup.tif",
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_NilStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CoverageStores.InWorkspace("ne").Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil coverage store") {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/workspaces/ne/coveragestores/states_tiff" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"coverageStore":{"enabled":false}}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	disabled := false
	err := c.CoverageStores.InWorkspace("ne").Update(context.Background(), "states_tiff",
		&coveragestores.Patch{Enabled: &disabled})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NilPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CoverageStores.InWorkspace("ne").Update(context.Background(), "states_tiff", nil)
	if err == nil || !strings.Contains(err.Error(), "nil patch") {
		t.Fatalf("expected nil-patch error, got %v", err)
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/workspaces/ne/coveragestores/states_tiff" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CoverageStores.InWorkspace("ne").Delete(context.Background(), "states_tiff",
		coveragestores.DeleteOptions{Recurse: true})
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
	err := c.CoverageStores.InWorkspace("ne").Delete(context.Background(), "states_tiff",
		coveragestores.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

func TestGet_URLEscaping(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"coverageStore":{"name":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.CoverageStores.InWorkspace("ws*1").Get(context.Background(), "cs*2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURI, "ws%2A1") || !strings.Contains(capturedURI, "cs%2A2") {
		t.Fatalf("expected single-encoded segments, got %q", capturedURI)
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}

func TestUploadFile_GeoTIFF(t *testing.T) {
	var captured struct {
		Method, Path, ContentType, Accept string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		captured.Accept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.CoverageStores.InWorkspace("nurc").UploadFile(context.Background(), "world_dem",
		strings.NewReader("FAKETIFF"),
		coveragestores.UploadOptions{Extension: "geotiff", ContentType: "image/tiff"}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if captured.Method != http.MethodPut {
		t.Errorf("Method = %q, want PUT", captured.Method)
	}
	if captured.Path != "/rest/workspaces/nurc/coveragestores/world_dem/file.geotiff" {
		t.Errorf("Path = %q", captured.Path)
	}
	if captured.ContentType != "image/tiff" {
		t.Errorf("Content-Type = %q, want image/tiff", captured.ContentType)
	}
	if captured.Accept != "*/*" {
		t.Errorf("Accept = %q, want */*", captured.Accept)
	}
}

func TestUploadFile_ImageMosaic(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.CoverageStores.InWorkspace("nurc").UploadFile(context.Background(), "mosaic",
		strings.NewReader("FAKEZIP"),
		coveragestores.UploadOptions{Extension: "imagemosaic"}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if capturedPath != "/rest/workspaces/nurc/coveragestores/mosaic/file.imagemosaic" {
		t.Errorf("Path = %q", capturedPath)
	}
}

func TestHarvestGranule_OK(t *testing.T) {
	var captured struct{ Method, Path, ContentType string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	// External method: pass a server-local path string as the body.
	if err := c.CoverageStores.InWorkspace("nurc").HarvestGranule(context.Background(), "mosaic",
		strings.NewReader("/srv/geoserver/granules/2026_05_03.tif"),
		coveragestores.UploadOptions{Method: coveragestores.UploadMethodExternal, Extension: "imagemosaic"}); err != nil {
		t.Fatalf("HarvestGranule: %v", err)
	}
	if captured.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", captured.Method)
	}
	if captured.Path != "/rest/workspaces/nurc/coveragestores/mosaic/external.imagemosaic" {
		t.Errorf("Path = %q", captured.Path)
	}
	if captured.ContentType != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", captured.ContentType)
	}
}

func TestUploadFile_Validation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	cases := []struct {
		name string
		ws   string
		cs   string
		body io.Reader
	}{
		{"empty workspace", "", "world_dem", strings.NewReader("")},
		{"empty name", "nurc", "", strings.NewReader("")},
		{"nil body", "nurc", "world_dem", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.CoverageStores.InWorkspace(tc.ws).UploadFile(context.Background(), tc.cs, tc.body,
				coveragestores.UploadOptions{Extension: "geotiff"})
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestUploadFile_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no workspace", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CoverageStores.InWorkspace("missing").UploadFile(context.Background(), "world_dem",
		strings.NewReader(""), coveragestores.UploadOptions{Extension: "geotiff"})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
