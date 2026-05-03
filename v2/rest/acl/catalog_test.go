package acl_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
)

func TestCatalog_Get_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/security/acl/catalog" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"mode":"HIDE"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	mode, err := c.ACL.Catalog().Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != acl.CatalogModeHide {
		t.Errorf("mode = %q, want HIDE", mode)
	}
}

func TestCatalog_Get_OtherModes(t *testing.T) {
	for _, m := range []acl.CatalogMode{acl.CatalogModeMixed, acl.CatalogModeChallenge} {
		t.Run(string(m), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"mode":"`+string(m)+`"}`)
			}))
			defer srv.Close()

			c := newTestClient(t, srv)
			got, err := c.ACL.Catalog().Get(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != m {
				t.Errorf("mode = %q, want %q", got, m)
			}
		})
	}
}

func TestCatalog_Update_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/security/acl/catalog" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"mode":"MIXED"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Catalog().Update(context.Background(), acl.CatalogModeMixed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCatalog_Update_Invalid(t *testing.T) {
	// GeoServer returns 422 for invalid catalog modes; the client
	// should surface that as a 4xx error (no specific sentinel).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Catalog().Update(context.Background(), acl.CatalogMode("BOGUS"))
	if err == nil {
		t.Fatalf("expected error on 422, got nil")
	}
	var apiErr *geoserver.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 APIError, got %v", err)
	}
}

func TestCatalog_Reload_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/acl/catalog/reload" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.ACL.Catalog().Reload(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
