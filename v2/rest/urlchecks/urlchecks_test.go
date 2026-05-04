package urlchecks_test

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
	"github.com/hishamkaram/geoserver/v2/rest/urlchecks"
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

func TestList_Empty_StringWireShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"urlChecks":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.URLChecks.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list for {urlChecks:\"\"}, got %d", len(list))
	}
}

func TestList_Populated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/urlchecks" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"urlChecks":{"urlCheck":[
			{"name":"icons","href":"http://srv/rest/urlchecks/icons.json"},
			{"name":"safeWFS","href":"http://srv/rest/urlchecks/safeWFS.json"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.URLChecks.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d", len(list))
	}
	if list[0].Name != "icons" || list[1].Name != "safeWFS" {
		t.Errorf("list = %+v", list)
	}
}

func TestGet_WrappedShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"regexUrlCheck":{"name":"icons","description":"External graphic icons","enabled":true,"regex":"^https://styles.example/.*$"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	chk, err := c.URLChecks.Get(context.Background(), "icons")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if chk.Name != "icons" || chk.Description != "External graphic icons" || !chk.Enabled || chk.Regex == "" {
		t.Errorf("check = %+v", chk)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.URLChecks.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreate_BodyIsRegexUrlCheckEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/urlchecks" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		// Body MUST be wrapped — flat JSON is rejected by the server.
		for _, want := range []string{`"regexUrlCheck":{`, `"name":"icons"`, `"regex":"^https://`, `"enabled":true`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.URLChecks.Create(context.Background(), &urlchecks.URLCheck{
		Name: "icons", Regex: "^https://example/.*$", Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_RequiresFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	for _, in := range []*urlchecks.URLCheck{nil, {}, {Name: "x"}, {Regex: "y"}} {
		err := c.URLChecks.Create(context.Background(), in)
		if err == nil {
			t.Errorf("expected error for input %+v", in)
		}
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/urlchecks/icons" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.URLChecks.Update(context.Background(), "icons", &urlchecks.URLCheck{Enabled: true})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/urlchecks/icons" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.URLChecks.Delete(context.Background(), "icons"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.URLChecks.Delete(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
