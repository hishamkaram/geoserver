package workspaces_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// newTestClient constructs a v2 *Client pointed at the given httptest
// server, with basic-auth and a 5s timeout.
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

// expectBasicAuth asserts the Authorization header carries the
// admin/geoserver basic-auth value.
func expectBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:geoserver"))
	if got := r.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization header = %q, want %q", got, want)
	}
}

// expectUserAgent asserts the User-Agent header is the default v2 UA.
func expectUserAgent(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("User-Agent"); got != "geoserver-go/v2" {
		t.Fatalf("User-Agent = %q, want %q", got, "geoserver-go/v2")
	}
}

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		expectUserAgent(t, r)
		if r.Method != http.MethodGet || r.URL.Path != "/rest/workspaces" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"workspaces":{"workspace":[{"name":"topp"},{"name":"sf","isolated":true}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Workspaces.List(context.Background(), workspaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d workspaces, want 2", len(got))
	}
	if got[0].Name != "topp" || got[1].Name != "sf" || !got[1].Isolated {
		t.Fatalf("unexpected workspaces: %+v", got)
	}
}

func TestList_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "bad creds")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Workspaces.List(context.Background(), workspaces.ListOptions{})
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	var apiErr *geoserver.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *geoserver.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("APIError.StatusCode = %d", apiErr.StatusCode)
	}
	if apiErr.Op != "Workspaces.List" {
		t.Fatalf("APIError.Op = %q, want Workspaces.List", apiErr.Op)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"workspaces":{"workspace":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for ws, err := range c.Workspaces.Iter(context.Background(), workspaces.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, ws.Name)
	}
	if len(names) != 3 || names[0] != "a" || names[2] != "c" {
		t.Fatalf("iterator yielded %v", names)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"workspace":{"name":"topp","isolated":false}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ws, err := c.Workspaces.Get(context.Background(), "topp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Name != "topp" {
		t.Fatalf("Name = %q", ws.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "no such workspace")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Workspaces.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/workspaces" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"workspace":{"name":"new_ws"}}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Workspaces.Create(context.Background(), &workspaces.Workspace{Name: "new_ws"}); err != nil {
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
	err := c.Workspaces.Create(context.Background(), &workspaces.Workspace{Name: "topp"})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/workspaces/topp" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"workspace":{"isolated":true}}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	isolated := true
	if err := c.Workspaces.Update(context.Background(), "topp", &workspaces.WorkspacePatch{Isolated: &isolated}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/workspaces/topp" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q, want true", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Workspaces.Delete(context.Background(), "topp", workspaces.DeleteOptions{Recurse: true}); err != nil {
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
	err := c.Workspaces.Delete(context.Background(), "topp", workspaces.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

// URL-escaping regression: workspace names with characters that
// PathEscape encodes to a sequence containing "%" must produce a
// single-encoded URL on the wire (not double-encoded "%25..").
func TestGet_URLEscaping_Asterisk(t *testing.T) {
	var capturedRequestURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"workspace":{"name":"weird"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Workspaces.Get(context.Background(), "weird*name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(capturedRequestURI, "weird%2Aname") {
		t.Fatalf("expected %%2A in request URI, got %q", capturedRequestURI)
	}
	if contains(capturedRequestURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedRequestURI)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
