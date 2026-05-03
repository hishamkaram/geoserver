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

// ---- ServiceRule encode / decode round-trip ----

func TestServiceRule_Encode_Defaults(t *testing.T) {
	r := acl.ServiceRule{}
	rule, roles := r.Encode()
	if rule != "*.*" {
		t.Errorf("rule = %q, want *.*", rule)
	}
	if roles != "*" {
		t.Errorf("roles = %q, want *", roles)
	}
}

func TestServiceRule_Encode_Full(t *testing.T) {
	r := acl.ServiceRule{
		Service:   "wms",
		Operation: "GetMap",
		Roles:     []string{"ROLE_USER", "ROLE_ADMIN"},
	}
	rule, roles := r.Encode()
	if rule != "wms.GetMap" {
		t.Errorf("rule = %q", rule)
	}
	if roles != "ROLE_USER,ROLE_ADMIN" {
		t.Errorf("roles = %q", roles)
	}
}

func TestDecodeServiceRule_OK(t *testing.T) {
	r, err := acl.DecodeServiceRule("wfs.GetFeature", "ROLE_READER")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.Service != "wfs" || r.Operation != "GetFeature" {
		t.Fatalf("Rule = %+v", r)
	}
	if len(r.Roles) != 1 || r.Roles[0] != "ROLE_READER" {
		t.Fatalf("Roles = %v", r.Roles)
	}
}

func TestDecodeServiceRule_BadFormat(t *testing.T) {
	_, err := acl.DecodeServiceRule("toomany.dots.here", "*")
	if err == nil || !strings.Contains(err.Error(), "service.operation") {
		t.Fatalf("expected format error, got %v", err)
	}
}

// ---- ServicesClient HTTP CRUD ----

func TestServices_List_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/acl/services" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"*.*": "*",
			"wms.GetMap": "ROLE_USER",
			"wfs.GetFeature": "ROLE_READER,ROLE_ADMIN"
		}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rules, err := c.ACL.Services().List(context.Background(), acl.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("len = %d, want 3; rules = %+v", len(rules), rules)
	}
	encoded := make([]string, 0, len(rules))
	for _, r := range rules {
		ruleStr, rolesStr := r.Encode()
		encoded = append(encoded, ruleStr+"="+rolesStr)
	}
	sort.Strings(encoded)
	want := []string{
		"*.*=*",
		"wfs.GetFeature=ROLE_READER,ROLE_ADMIN",
		"wms.GetMap=ROLE_USER",
	}
	for i, w := range want {
		if encoded[i] != w {
			t.Errorf("encoded[%d] = %q, want %q", i, encoded[i], w)
		}
	}
}

func TestServices_Add_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/acl/services" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"wms.GetMap":"ROLE_USER"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Services().Add(context.Background(), acl.ServiceRule{
		Service: "wms", Operation: "GetMap", Roles: []string{"ROLE_USER"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServices_Add_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Services().Add(context.Background(), acl.ServiceRule{
		Service: "wms", Operation: "GetMap",
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestServices_Add_EmptyServiceAndOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Services().Add(context.Background(), acl.ServiceRule{})
	if err == nil || !strings.Contains(err.Error(), "empty Service and Operation") {
		t.Fatalf("expected empty-service-and-operation error, got %v", err)
	}
}

func TestServices_Update_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/security/acl/services" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Services().Update(context.Background(), acl.ServiceRule{
		Service: "wms", Operation: "GetMap", Roles: []string{"ROLE_ADMIN"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServices_Delete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/security/acl/services/wms.GetMap" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Services().Delete(context.Background(), acl.ServiceRule{
		Service: "wms", Operation: "GetMap",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServices_Delete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.ACL.Services().Delete(context.Background(), acl.ServiceRule{
		Service: "wms", Operation: "GetMap",
	})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
