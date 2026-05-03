package about_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestPing_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/about/version" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"about":{"resource":[]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.About.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPing_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.About.Ping(context.Background())
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPing_ServerDown(t *testing.T) {
	// Construct a client pointing at a port nothing's listening on.
	c, err := geoserver.New("http://127.0.0.1:1",
		geoserver.WithTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = c.About.Ping(context.Background())
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	// Transport errors are not wrapped as *APIError — they're raw
	// net.OpError equivalents.
	var apiErr *geoserver.APIError
	if errors.As(err, &apiErr) {
		t.Fatalf("transport error should not be *APIError, got %v", apiErr)
	}
}

func TestVersion_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"about":{"resource":[
			{"@name":"GeoServer","Version":"2.28.0","Build-Timestamp":"03-May-2026 10:00","Git-Revision":"abc123"},
			{"@name":"GeoTools","Version":"32.0"},
			{"@name":"GeoWebCache","Version":"1.28.0"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	v, err := c.About.Version(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Resource) != 3 {
		t.Fatalf("Resource len = %d, want 3", len(v.Resource))
	}
	if v.Resource[0].Name != "GeoServer" || v.Resource[0].Version != "2.28.0" {
		t.Fatalf("Resource[0] = %+v", v.Resource[0])
	}
	if v.Resource[0].BuildTimestamp == "" || v.Resource[0].GitRevision != "abc123" {
		t.Fatalf("Resource[0] missing build info: %+v", v.Resource[0])
	}
}

func TestVersion_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.About.Version(context.Background())
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}
