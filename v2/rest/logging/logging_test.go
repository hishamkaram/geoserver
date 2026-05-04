package logging_test

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
	"github.com/hishamkaram/geoserver/v2/rest/logging"
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

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/logging" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"logging":{"level":"DEFAULT_LOGGING","location":"logs/geoserver.log","stdOutLogging":true}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	cfg, err := c.Logging.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg.Level != "DEFAULT_LOGGING" || cfg.Location != "logs/geoserver.log" || !cfg.StdOutLogging {
		t.Errorf("config = %+v", cfg)
	}
}

func TestUpdate_BodyWrappedInLoggingEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/logging" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, want := range []string{`"logging":{`, `"level":"VERBOSE_LOGGING"`, `"stdOutLogging":true`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Logging.Update(context.Background(), &logging.Config{
		Level:         "VERBOSE_LOGGING",
		StdOutLogging: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestUpdate_NilRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Logging.Update(context.Background(), nil); err == nil {
		t.Fatal("expected nil-config error")
	}
}

func TestGet_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Logging.Get(context.Background())
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
