package styles_test

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
	"github.com/hishamkaram/geoserver/v2/rest/styles"
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

// ---- Scope accessors ----

func TestScope_GlobalDefault(t *testing.T) {
	c, err := geoserver.New("http://localhost:8080/geoserver")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !c.Styles.IsGlobal() || c.Styles.Workspace() != "" {
		t.Fatalf("default scope should be global; got Workspace=%q IsGlobal=%v",
			c.Styles.Workspace(), c.Styles.IsGlobal())
	}
}

func TestScope_InWorkspace(t *testing.T) {
	c, err := geoserver.New("http://localhost:8080/geoserver")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ws := c.Styles.InWorkspace("topp")
	if ws.IsGlobal() || ws.Workspace() != "topp" {
		t.Fatalf("InWorkspace scope wrong: %+v", ws)
	}
	// Original client unchanged.
	if !c.Styles.IsGlobal() {
		t.Fatalf("InWorkspace mutated the original client")
	}
}

// ---- List with empty-collection wire quirk ----

func TestList_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/styles" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// GeoServer 2.28's empty-collection shape.
		_, _ = io.WriteString(w, `{"styles":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Styles.List(context.Background(), styles.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for empty collection, got %+v", got)
	}
}

func TestList_Populated_Global(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.URL.Path != "/rest/styles" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"styles":{"style":[{"name":"polygon","filename":"polygon.sld"},{"name":"line"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Styles.List(context.Background(), styles.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "polygon" || got[0].Filename != "polygon.sld" {
		t.Fatalf("List = %+v", got)
	}
}

func TestList_Populated_Workspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/styles" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"styles":{"style":[{"name":"local"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Styles.InWorkspace("topp").List(context.Background(), styles.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "local" {
		t.Fatalf("List = %+v", got)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"styles":{"style":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for s, err := range c.Styles.Iter(context.Background(), styles.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		names = append(names, s.Name)
	}
	if len(names) != 3 {
		t.Fatalf("Iter = %v", names)
	}
}

// ---- Get ----

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/styles/polygon" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"style":{"name":"polygon","format":"sld","filename":"polygon.sld","languageVersion":{"version":"1.0.0"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	s, err := c.Styles.Get(context.Background(), "polygon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "polygon" || s.Format != "sld" || s.Filename != "polygon.sld" {
		t.Fatalf("Style = %+v", s)
	}
	if s.LanguageVersion == nil || s.LanguageVersion.Version != "1.0.0" {
		t.Fatalf("LanguageVersion = %+v", s.LanguageVersion)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Styles.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ---- Create — global path uses default Accept; workspace path forces "*/*" ----

func TestCreate_Global_DefaultAccept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/styles" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		// Default Accept is application/json on global path.
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"name":"polygon"`) ||
			!strings.Contains(string(body), `"filename":"polygon.sld"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Styles.Create(context.Background(), &styles.Style{Name: "polygon"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Workspace_AcceptStarStar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/workspaces/topp/styles" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		// Workspace-scoped quirk: Accept must be */*.
		if got := r.Header.Get("Accept"); got != "*/*" {
			t.Errorf("Accept = %q, want */* (workspace-scoped POST quirk)", got)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.InWorkspace("topp").Create(context.Background(), &styles.Style{Name: "local"})
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
	err := c.Styles.Create(context.Background(), &styles.Style{Name: "dup"})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_DefaultsFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Filename was empty on input — Create must default to "{name}.sld".
		if !strings.Contains(string(body), `"filename":"states.sld"`) {
			t.Errorf("expected default filename in body: %s", body)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Styles.Create(context.Background(), &styles.Style{Name: "states"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_NilStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil style") {
		t.Fatalf("expected nil-style error, got %v", err)
	}
}

// ---- UploadSLD: raw body + content-type ----

func TestUploadSLD_DefaultContentType(t *testing.T) {
	const sldBody = `<?xml version="1.0" encoding="UTF-8"?><StyledLayerDescriptor xmlns="http://www.opengis.net/sld" version="1.0.0"></StyledLayerDescriptor>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/styles/polygon" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/vnd.ogc.sld+xml" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != sldBody {
			t.Errorf("body roundtrip mismatch:\nwant %q\ngot  %q", sldBody, body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.UploadSLD(context.Background(), "polygon",
		strings.NewReader(sldBody), styles.UploadOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadSLD_CustomFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/vnd.geoserver.geocss+css" {
			t.Errorf("Content-Type = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.UploadSLD(context.Background(), "polygon",
		strings.NewReader("* { stroke: red; }"),
		styles.UploadOptions{Format: "application/vnd.geoserver.geocss+css"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadSLD_NilBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.UploadSLD(context.Background(), "polygon", nil, styles.UploadOptions{})
	if err == nil || !strings.Contains(err.Error(), "nil body") {
		t.Fatalf("expected nil-body error, got %v", err)
	}
}

func TestUploadSLD_400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "invalid SLD")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.UploadSLD(context.Background(), "polygon",
		strings.NewReader("not really sld"), styles.UploadOptions{})
	if !errors.Is(err, geoserver.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// ---- Update ----

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/styles/polygon" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"format":"sld"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.Update(context.Background(), "polygon", &styles.Style{Format: "sld"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Delete: ?purge= ----

func TestDelete_PurgeQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/styles/polygon" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("purge") != "true" {
			t.Errorf("purge = %q", r.URL.Query().Get("purge"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.Delete(context.Background(), "polygon",
		styles.DeleteOptions{Purge: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NoPurge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("purge") != "false" {
			t.Errorf("purge = %q, want false", r.URL.Query().Get("purge"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Styles.Delete(context.Background(), "polygon", styles.DeleteOptions{})
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
	err := c.Styles.Delete(context.Background(), "polygon", styles.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

// ---- URL escaping ----

func TestGet_URLEscaping_Workspace(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"style":{"name":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Styles.InWorkspace("ws*1").Get(context.Background(), "st*2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURI, "ws%2A1") || !strings.Contains(capturedURI, "st%2A2") {
		t.Fatalf("expected single-encoded segments, got %q", capturedURI)
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}
