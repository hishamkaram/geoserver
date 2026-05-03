package resources_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/resources"
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

// ---- Get (file content streaming) ----

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/resource/styles/default_point.sld" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("operation") != "default" {
			t.Errorf("operation = %q", r.URL.Query().Get("operation"))
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = io.WriteString(w, `<sld>example</sld>`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body, err := c.Resources.Get(context.Background(), "styles/default_point.sld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != `<sld>example</sld>` {
		t.Errorf("body = %q", got)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resources.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Path with leading slash should still produce the same URL.
func TestGet_LeadingSlashOK(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/rest/resource/styles/foo.sld" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = io.WriteString(w, "x")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body, err := c.Resources.Get(context.Background(), "/styles/foo.sld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body.Close()
	if !called {
		t.Fatal("server not hit")
	}
}

// Path segments with spaces / non-ASCII must be URL-encoded.
func TestGet_PathEscaping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decoded form on the server side is "/rest/resource/styles/with space.sld".
		if r.URL.Path != "/rest/resource/styles/with space.sld" {
			t.Errorf("decoded path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = io.WriteString(w, "x")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body, err := c.Resources.Get(context.Background(), "styles/with space.sld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body.Close()
}

// ---- Stat (metadata) ----

func TestStat_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("operation") != "metadata" || r.URL.Query().Get("format") != "json" {
			t.Errorf("operation=%q format=%q", r.URL.Query().Get("operation"), r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceMetadata":{"name":"default_point.sld","parent":{"path":"/styles","link":{"href":"http://srv/rest/resource/styles","rel":"alternate","type":"application/json"}},"lastModified":"2025-10-13 05:03:12.0 UTC","type":"resource"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	meta, err := c.Resources.Stat(context.Background(), "styles/default_point.sld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "default_point.sld" {
		t.Errorf("Name = %q", meta.Name)
	}
	if meta.ParentPath != "/styles" {
		t.Errorf("ParentPath = %q", meta.ParentPath)
	}
	if meta.Type != resources.TypeResource {
		t.Errorf("Type = %q", meta.Type)
	}
	if meta.LastModified != "2025-10-13 05:03:12.0 UTC" {
		t.Errorf("LastModified = %q", meta.LastModified)
	}
}

func TestStat_UndefinedTypeMapsToNotFound(t *testing.T) {
	// GeoServer's wire-quirk: missing path returns 200 + type="undefined"
	// instead of 404. The SDK translates this into ErrNotFound.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceMetadata":{"name":"missing.sld","parent":{"path":"styles"},"lastModified":"1970-01-01 00:00:00.0 UTC","type":"undefined"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Resources.Stat(context.Background(), "styles/missing.sld")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for type=undefined, got %v", err)
	}
}

func TestStat_Directory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceMetadata":{"name":"styles","parent":{"path":"/","link":{"href":"http://srv/rest/resource/","rel":"alternate","type":"application/json"}},"lastModified":"2025-10-13 05:03:12.0 UTC","type":"directory"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	meta, err := c.Resources.Stat(context.Background(), "styles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Type != resources.TypeDirectory {
		t.Errorf("Type = %q, want directory", meta.Type)
	}
}

// ---- List (directory listing) ----

func TestList_OK_MultipleChildren(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("operation") != "default" || r.URL.Query().Get("format") != "json" {
			t.Errorf("operation=%q format=%q", r.URL.Query().Get("operation"), r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceDirectory":{"name":"styles","parent":{"path":"","link":{"href":"http://srv/rest/resource/","rel":"alternate","type":"application/json"}},"lastModified":"2025-10-13 05:03:12.0 UTC","children":{"child":[{"name":"a.sld","link":{"href":"http://srv/rest/resource/styles/a.sld","rel":"alternate","type":"text/xml"}},{"name":"b.png","link":{"href":"http://srv/rest/resource/styles/b.png","rel":"alternate","type":"image/png"}}]}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	dir, err := c.Resources.List(context.Background(), "styles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir.Name != "styles" || dir.Type != resources.TypeDirectory {
		t.Errorf("dir = %+v", dir.Metadata)
	}
	if len(dir.Children) != 2 {
		t.Fatalf("len(Children) = %d", len(dir.Children))
	}
	if dir.Children[0].Name != "a.sld" || dir.Children[0].MimeType != "text/xml" {
		t.Errorf("child[0] = %+v", dir.Children[0])
	}
	if dir.Children[1].Name != "b.png" || dir.Children[1].MimeType != "image/png" {
		t.Errorf("child[1] = %+v", dir.Children[1])
	}
}

// Children may collapse to a single object (not array) per GeoServer's
// classic single-element wire shape.
func TestList_OK_SingleChildCollapsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceDirectory":{"name":"templates","parent":{"path":""},"lastModified":"2025-10-13","children":{"child":{"name":"only.ftl","link":{"href":"http://srv/rest/resource/templates/only.ftl","rel":"alternate","type":"text/plain"}}}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	dir, err := c.Resources.List(context.Background(), "templates")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dir.Children) != 1 {
		t.Fatalf("expected 1 child via collapse, got %d", len(dir.Children))
	}
	if dir.Children[0].Name != "only.ftl" {
		t.Errorf("child = %+v", dir.Children[0])
	}
}

// Children may be an empty string for an empty directory — same
// pattern GeoServer uses for empty datastore / coverage / styles
// collections elsewhere.
func TestList_OK_EmptyChildrenAsString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceDirectory":{"name":"empty","parent":{"path":""},"lastModified":"2025-10-13","children":{"child":""}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	dir, err := c.Resources.List(context.Background(), "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dir.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(dir.Children))
	}
}

// ---- Exists ----

func TestExists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ResourceMetadata":{"name":"x.sld","parent":{"path":""},"lastModified":"2025-10-13","type":"resource"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exists, typ, err := c.Resources.Exists(context.Background(), "x.sld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("expected exists=true")
	}
	if typ != resources.TypeResource {
		t.Errorf("type = %q", typ)
	}
}

func TestExists_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exists, typ, err := c.Resources.Exists(context.Background(), "x.sld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Errorf("expected exists=false")
	}
	if typ != "" {
		t.Errorf("type = %q on missing, want empty", typ)
	}
}

// ---- Put ----

func TestPut_Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/resource/templates/foo.ftl" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("operation") != "default" {
			t.Errorf("operation = %q", r.URL.Query().Get("operation"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/plain" {
			t.Errorf("Content-Type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" {
			t.Errorf("body = %q", body)
		}
		w.WriteHeader(http.StatusCreated) // 201 — new resource
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Put(context.Background(), "templates/foo.ftl",
		strings.NewReader("hello"), "text/plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPut_OverwriteOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK) // 200 — existing resource overwritten
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Put(context.Background(), "templates/foo.ftl",
		strings.NewReader("hello"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPut_RootPathRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	for _, p := range []string{"", "/"} {
		err := c.Resources.Put(context.Background(), p, strings.NewReader("x"), "")
		if err == nil || !strings.Contains(err.Error(), "non-root") {
			t.Errorf("path %q: expected non-root error, got %v", p, err)
		}
	}
}

// ---- Move ----

func TestMove_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/resource/templates/bar.ftl" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("operation") != "move" {
			t.Errorf("operation = %q", r.URL.Query().Get("operation"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "templates/foo.ftl" {
			t.Errorf("body = %q (expected source path)", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Move(context.Background(), "templates/foo.ftl", "templates/bar.ftl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMove_LeadingSlashesStripped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "templates/foo.ftl" {
			t.Errorf("body = %q (leading slash should be stripped)", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Move(context.Background(), "/templates/foo.ftl", "/templates/bar.ftl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Copy ----

func TestCopy_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("operation") != "copy" {
			t.Errorf("operation = %q", r.URL.Query().Get("operation"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Copy(context.Background(), "templates/foo.ftl", "templates/foo_backup.ftl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Delete ----

func TestDelete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/resource/templates/foo.ftl" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Delete(context.Background(), "templates/foo.ftl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Resources.Delete(context.Background(), "missing.ftl")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_RootPathRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	for _, p := range []string{"", "/"} {
		err := c.Resources.Delete(context.Background(), p)
		if err == nil || !strings.Contains(err.Error(), "root resource") {
			t.Errorf("path %q: expected root-rejection error, got %v", p, err)
		}
	}
}
