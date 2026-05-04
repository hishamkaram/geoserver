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
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
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

// ---- Encode / DecodeRule round-trip ----

func TestRule_Encode_Defaults(t *testing.T) {
	r := acl.Rule{} // all empty
	rule, roles := r.Encode()
	if rule != "*.*.r" {
		t.Errorf("rule = %q, want *.*.r", rule)
	}
	if roles != "*" {
		t.Errorf("roles = %q, want *", roles)
	}
}

func TestRule_Encode_Full(t *testing.T) {
	r := acl.Rule{
		Workspace: "topp", Layer: "states",
		Operation: acl.OpWrite,
		Roles:     []string{"ROLE_EDITOR", "ROLE_ADMIN"},
	}
	rule, roles := r.Encode()
	if rule != "topp.states.w" {
		t.Errorf("rule = %q", rule)
	}
	if roles != "ROLE_EDITOR,ROLE_ADMIN" {
		t.Errorf("roles = %q", roles)
	}
}

func TestDecodeRule_OK(t *testing.T) {
	r, err := acl.DecodeRule("topp.states.w", "ROLE_EDITOR,ROLE_ADMIN")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.Workspace != "topp" || r.Layer != "states" || r.Operation != acl.OpWrite {
		t.Fatalf("Rule = %+v", r)
	}
	if len(r.Roles) != 2 || r.Roles[0] != "ROLE_EDITOR" {
		t.Fatalf("Roles = %v", r.Roles)
	}
}

func TestDecodeRule_AnyRoles(t *testing.T) {
	for _, rolesStr := range []string{"", "*"} {
		r, err := acl.DecodeRule("*.*.r", rolesStr)
		if err != nil {
			t.Fatalf("decode %q: %v", rolesStr, err)
		}
		if r.Workspace != "*" || r.Layer != "*" {
			t.Errorf("Rule = %+v", r)
		}
		if len(r.Roles) != 0 {
			t.Errorf("expected empty Roles for input %q, got %v", rolesStr, r.Roles)
		}
	}
}

func TestDecodeRule_BadFormat(t *testing.T) {
	_, err := acl.DecodeRule("not.dotted", "*")
	if err == nil || !strings.Contains(err.Error(), "workspace.layer.op") {
		t.Fatalf("expected format error, got %v", err)
	}
}

// ---- HTTP CRUD ----

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/acl/layers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"topp.states.r": "*",
			"topp.states.w": "ROLE_EDITOR",
			"sf.archsites.a": "ROLE_ADMIN,ROLE_OWNER"
		}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rules, err := c.ACL.Layers().List(context.Background(), acl.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("len = %d, want 3; rules = %+v", len(rules), rules)
	}

	// Map iteration is unordered; collect encoded form for stable comparison.
	encoded := make([]string, 0, len(rules))
	for _, r := range rules {
		ruleStr, rolesStr := r.Encode()
		encoded = append(encoded, ruleStr+"="+rolesStr)
	}
	sort.Strings(encoded)
	want := []string{
		"sf.archsites.a=ROLE_ADMIN,ROLE_OWNER",
		"topp.states.r=*",
		"topp.states.w=ROLE_EDITOR",
	}
	for i, w := range want {
		if encoded[i] != w {
			t.Errorf("encoded[%d] = %q, want %q", i, encoded[i], w)
		}
	}
}

func TestList_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rules, err := c.ACL.Layers().List(context.Background(), acl.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected empty, got %+v", rules)
	}
}

func TestList_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ACL.Layers().List(context.Background(), acl.ListOptions{})
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAdd_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/acl/layers" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		// Body should be a single-key map with the encoded rule.
		if !strings.Contains(s, `"topp.states.w":"ROLE_EDITOR"`) {
			t.Errorf("body = %s", s)
		}
		// GeoServer returns 200 (not 201) for ACL adds.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Layers().Add(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states",
		Operation: acl.OpWrite, Roles: []string{"ROLE_EDITOR"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdd_AnyRolesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Empty Roles slice should encode to "*".
		if !strings.Contains(string(body), `"topp.states.r":"*"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Layers().Add(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states", Operation: acl.OpRead,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdd_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Layers().Add(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states", Operation: acl.OpRead,
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestAdd_EmptyOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Layers().Add(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states", // Operation missing
	})
	if err == nil || !strings.Contains(err.Error(), "empty Operation") {
		t.Fatalf("expected empty-operation error, got %v", err)
	}
}

func TestDelete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/security/acl/layers/topp.states.w" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Layers().Delete(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states", Operation: acl.OpWrite,
	})
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
	err := c.ACL.Layers().Delete(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states", Operation: acl.OpRead,
	})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_EmptyOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Layers().Delete(context.Background(), acl.Rule{
		Workspace: "topp", Layer: "states",
	})
	if err == nil || !strings.Contains(err.Error(), "empty Operation") {
		t.Fatalf("expected empty-operation error, got %v", err)
	}
}
