package namespaces_test

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
	"github.com/hishamkaram/geoserver/v2/rest/namespaces"
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

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/namespaces" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"namespaces":{"namespace":[{"prefix":"topp","uri":"http://www.openplans.org/topp"},{"prefix":"sf","uri":"http://cite.opengeospatial.org/gmlsf"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Namespaces.List(context.Background(), namespaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Prefix != "topp" || got[1].URI != "http://cite.opengeospatial.org/gmlsf" {
		t.Fatalf("List = %+v", got)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"namespaces":{"namespace":[{"prefix":"a"},{"prefix":"b"},{"prefix":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var prefixes []string
	for n, err := range c.Namespaces.Iter(context.Background(), namespaces.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		prefixes = append(prefixes, n.Prefix)
	}
	if len(prefixes) != 3 {
		t.Fatalf("Iter = %v", prefixes)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/namespaces/topp" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"namespace":{"prefix":"topp","uri":"http://www.openplans.org/topp","isolated":false}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ns, err := c.Namespaces.Get(context.Background(), "topp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ns.Prefix != "topp" || ns.URI != "http://www.openplans.org/topp" || ns.Isolated {
		t.Fatalf("Namespace = %+v", ns)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Namespaces.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_EmptyPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Namespaces.Get(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "empty prefix") {
		t.Fatalf("expected empty-prefix error, got %v", err)
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/namespaces" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, sub := range []string{
			`"namespace":`,
			`"prefix":"ne"`,
			`"uri":"http://example.com/ne"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Namespaces.Create(context.Background(), &namespaces.Namespace{
		Prefix: "ne", URI: "http://example.com/ne",
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
	err := c.Namespaces.Create(context.Background(), &namespaces.Namespace{
		Prefix: "dup", URI: "http://example.com/dup",
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_NilNamespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Namespaces.Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil namespace") {
		t.Fatalf("expected nil-namespace error, got %v", err)
	}
}

func TestCreate_EmptyURI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Namespaces.Create(context.Background(), &namespaces.Namespace{Prefix: "ne"})
	if err == nil || !strings.Contains(err.Error(), "empty URI") {
		t.Fatalf("expected empty-URI error, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/namespaces/topp" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"namespace":{"uri":"http://new.example.com/topp"}}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	uri := "http://new.example.com/topp"
	err := c.Namespaces.Update(context.Background(), "topp", &namespaces.Patch{URI: &uri})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/namespaces/topp" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Namespaces.Delete(context.Background(), "topp")
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
	err := c.Namespaces.Delete(context.Background(), "topp")
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

func TestGet_URLEscaping(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"namespace":{"prefix":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Namespaces.Get(context.Background(), "ns*1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURI, "ns%2A1") {
		t.Fatalf("expected single-encoded segment, got %q", capturedURI)
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}
