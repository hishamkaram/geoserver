package acl_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
)

// ---- RESTRule encode / decode round-trip ----

func TestRESTRule_Encode_Defaults(t *testing.T) {
	r := acl.RESTRule{}
	rule, roles := r.Encode()
	if rule != "/**:*" {
		t.Errorf("rule = %q, want /**:*", rule)
	}
	if roles != "*" {
		t.Errorf("roles = %q, want *", roles)
	}
}

func TestRESTRule_Encode_Full(t *testing.T) {
	r := acl.RESTRule{
		Pattern: "/rest/workspaces/**",
		Methods: []string{"POST", "PUT", "DELETE"},
		Roles:   []string{"ROLE_ADMIN"},
	}
	rule, roles := r.Encode()
	if rule != "/rest/workspaces/**:POST,PUT,DELETE" {
		t.Errorf("rule = %q", rule)
	}
	if roles != "ROLE_ADMIN" {
		t.Errorf("roles = %q", roles)
	}
}

func TestRESTRule_EncodePathSegment_SemicolonSeparator(t *testing.T) {
	r := acl.RESTRule{Pattern: "/**", Methods: []string{"GET"}}
	got := r.EncodePathSegment()
	if got != "/**;GET" {
		t.Errorf("EncodePathSegment = %q, want /**;GET", got)
	}
}

func TestDecodeRESTRule_AcceptsBothSeparators(t *testing.T) {
	for _, s := range []string{"/**:GET", "/**;GET"} {
		r, err := acl.DecodeRESTRule(s, "ROLE_ADMIN")
		if err != nil {
			t.Fatalf("decode %q: %v", s, err)
		}
		if r.Pattern != "/**" || len(r.Methods) != 1 || r.Methods[0] != "GET" {
			t.Errorf("decode %q -> %+v", s, r)
		}
	}
}

func TestDecodeRESTRule_BadFormat(t *testing.T) {
	_, err := acl.DecodeRESTRule("noseparator", "*")
	if err == nil || !strings.Contains(err.Error(), "pattern:methods") {
		t.Fatalf("expected format error, got %v", err)
	}
}

// ---- RESTClient HTTP CRUD ----

func TestREST_List_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/acl/rest" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"/**:GET": "*",
			"/**:POST,DELETE,PUT": "ADMIN"
		}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rules, err := c.ACL.REST().List(context.Background(), acl.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("len = %d, want 2", len(rules))
	}
	encoded := make([]string, 0, len(rules))
	for _, r := range rules {
		ruleStr, rolesStr := r.Encode()
		encoded = append(encoded, ruleStr+"="+rolesStr)
	}
	sort.Strings(encoded)
	want := []string{
		"/**:GET=*",
		"/**:POST,DELETE,PUT=ADMIN",
	}
	for i, w := range want {
		if encoded[i] != w {
			t.Errorf("encoded[%d] = %q, want %q", i, encoded[i], w)
		}
	}
}

func TestREST_Add_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/acl/rest" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"/**:GET":"*"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.REST().Add(context.Background(), acl.RESTRule{
		Pattern: "/**", Methods: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestREST_Add_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.REST().Add(context.Background(), acl.RESTRule{
		Pattern: "/**", Methods: []string{"GET"},
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestREST_Update_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/security/acl/rest" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.REST().Update(context.Background(), acl.RESTRule{
		Pattern: "/**", Methods: []string{"GET"}, Roles: []string{"ROLE_USER"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestREST_Delete_OK_LiteralSlashesAndSemicolon(t *testing.T) {
	// GeoServer requires the slashes, "*" globs, and ";" separator
	// to be transmitted literally — not percent-encoded.
	// The expected wire path is /rest/security/acl/rest//**;GET
	// (note the double-slash because the rule itself starts with /).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/rest/security/acl/rest//**;GET" {
			t.Errorf("path = %q, want /rest/security/acl/rest//**;GET", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.REST().Delete(context.Background(), acl.RESTRule{
		Pattern: "/**", Methods: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestREST_Delete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.REST().Delete(context.Background(), acl.RESTRule{
		Pattern: "/**", Methods: []string{"GET"},
	})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestREST_Add_EmptyPatternAndMethods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.REST().Add(context.Background(), acl.RESTRule{})
	if err == nil || !strings.Contains(err.Error(), "empty Pattern and Methods") {
		t.Fatalf("expected empty-pattern-and-methods error, got %v", err)
	}
}
