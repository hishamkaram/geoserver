package templates_test

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

// ---- List shapes ----

func TestList_Global_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/templates.json" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"org.geoserver.rest.catalog.TemplateInfos":{"org.geoserver.rest.catalog.TemplateInfo":[
			{"name":"a.ftl","href":"http://srv/rest/templates/a.ftl.json"},
			{"name":"b.ftl","href":"http://srv/rest/templates/b.ftl.json"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.Templates.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].Name != "a.ftl" {
		t.Errorf("list = %+v", list)
	}
}

func TestList_Workspace_PathOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/templates.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"org.geoserver.rest.catalog.TemplateInfos":{"org.geoserver.rest.catalog.TemplateInfo":[]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Templates.InWorkspace("topp").List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestList_FeatureType_PathOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/rest/workspaces/topp/datastores/states_pg/featuretypes/states/templates.json"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"org.geoserver.rest.catalog.TemplateInfos":{"org.geoserver.rest.catalog.TemplateInfo":[]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Templates.InWorkspace("topp").
		InDatastore("states_pg").
		InFeatureType("states").
		List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestList_Coverage_PathOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/rest/workspaces/nurc/coveragestores/mosaic/coverages/mosaic/templates.json"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"org.geoserver.rest.catalog.TemplateInfos":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Templates.InWorkspace("nurc").
		InCoverageStore("mosaic").
		InCoverage("mosaic").
		List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

// ---- Get / Put / Delete ----

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/templates/header.ftl" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "<#-- header -->\nHello")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body, err := c.Templates.Get(context.Background(), "header")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(body, "Hello") {
		t.Errorf("body = %q", body)
	}
}

func TestGet_NameWithFTLNotDoubled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Caller passed "header.ftl"; URL must NOT be header.ftl.ftl.
		if r.URL.Path != "/rest/templates/header.ftl" {
			t.Errorf("path = %q (suffix doubled?)", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "x")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Templates.Get(context.Background(), "header.ftl")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Templates.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPut_Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/templates/foo.ftl" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/plain" {
			t.Errorf("Content-Type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" {
			t.Errorf("body = %q", body)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Templates.PutString(context.Background(), "foo", "hello"); err != nil {
		t.Fatalf("PutString: %v", err)
	}
}

func TestPut_OverwriteOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Templates.PutString(context.Background(), "foo", "hello"); err != nil {
		t.Fatalf("PutString: %v", err)
	}
}

func TestPut_EmptyNameRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Templates.PutString(context.Background(), "", "x")
	if err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Fatalf("expected empty-name error, got %v", err)
	}
}

func TestDelete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/templates/foo.ftl" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Templates.Delete(context.Background(), "foo"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Templates.Delete(context.Background(), "foo")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ---- Workspace-scoped CRUD path verification ----

func TestPut_Workspace_PathOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/templates/content.ftl" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Templates.InWorkspace("topp").PutString(context.Background(), "content", "x")
	if err != nil {
		t.Fatalf("PutString: %v", err)
	}
}
