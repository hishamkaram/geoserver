package system_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

func TestReload_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/rest/reload" {
			t.Errorf("Path = %q, want /rest/reload", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.System.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
}

func TestReload_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "auth required", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("bad", "creds"))

	err := c.System.Reload(context.Background())
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestReload_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "non-admin user", http.StatusForbidden)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	err := c.System.Reload(context.Background())
	if !errors.Is(err, geoserver.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestResetCache_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/rest/reset" {
			t.Errorf("Path = %q, want /rest/reset", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	if err := c.System.ResetCache(context.Background()); err != nil {
		t.Fatalf("ResetCache: %v", err)
	}
}

func TestResetCache_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "kaboom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	err := c.System.ResetCache(context.Background())
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("err = %v, want ErrServerError", err)
	}
}
