package geoserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACLRule_ToStrings_Defaults(t *testing.T) {
	r := ACLRule{}
	rule, roles := r.ToStrings()
	assert.Equal(t, "*.*.r", rule)
	assert.Equal(t, "*", roles)
}

func TestACLRule_ToStrings_Populated(t *testing.T) {
	r := ACLRule{Workspace: "topp", Layer: "states", Operation: ACLOpWrite, Roles: []string{"editor", "admin"}}
	rule, roles := r.ToStrings()
	assert.Equal(t, "topp.states.w", rule)
	assert.Equal(t, "editor,admin", roles)
}

func TestStringToACLRule_OK(t *testing.T) {
	r, err := StringToACLRule("topp.states.r", "viewer,editor")
	assert.NoError(t, err)
	assert.Equal(t, "topp", r.Workspace)
	assert.Equal(t, "states", r.Layer)
	assert.Equal(t, ACLOpRead, r.Operation)
	assert.Equal(t, []string{"viewer", "editor"}, r.Roles)
}

func TestStringToACLRule_BadShape(t *testing.T) {
	_, err := StringToACLRule("not-a-rule", "")
	if err == nil {
		t.Fatalf("expected error for malformed rule")
	}
}

func TestACL_GetLayersACLRules_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/security/acl/layers", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"topp.states.r":"viewer","*.*.w":"admin"}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	rules, err := gs.GetLayersACLRulesContext(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rules, 2)

	// Two-rule response: order is map-iteration so we look up by content.
	var seenStates, seenAdmin bool
	for _, rule := range rules {
		if rule.Workspace == "topp" && rule.Layer == "states" {
			seenStates = true
			assert.Equal(t, ACLOpRead, rule.Operation)
			assert.Equal(t, []string{"viewer"}, rule.Roles)
		}
		if rule.Workspace == "*" && rule.Layer == "*" {
			seenAdmin = true
			assert.Equal(t, ACLOpWrite, rule.Operation)
		}
	}
	assert.True(t, seenStates)
	assert.True(t, seenAdmin)
}

func TestACL_GetLayersACLRules_403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "denied")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetLayersACLRulesContext(context.Background())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestACL_AddLayersACLRule_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/security/acl/layers", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		// Expect JSON body of the form {"topp.states.r":"viewer"}.
		assert.Contains(t, string(body), `"topp.states.r":"viewer"`)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	rule := ACLRule{Workspace: "topp", Layer: "states", Operation: ACLOpRead, Roles: []string{"viewer"}}
	added, err := gs.AddLayersACLRuleContext(context.Background(), rule)
	assert.NoError(t, err)
	assert.True(t, added)
}

func TestACL_AddLayersACLRule_409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "rule exists")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	added, err := gs.AddLayersACLRuleContext(context.Background(), ACLRule{Workspace: "topp", Layer: "states", Operation: ACLOpRead, Roles: []string{"viewer"}})
	assert.False(t, added)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestACL_DeleteLayersACLRule_URLPath(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.RawPath
		if capturedPath == "" {
			capturedPath = r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteLayersACLRuleContext(context.Background(), ACLRule{Workspace: "topp", Layer: "states", Operation: ACLOpRead})
	assert.NoError(t, err)
	assert.True(t, deleted)
	if !strings.HasSuffix(capturedPath, "/rest/security/acl/layers/topp.states.r") {
		t.Fatalf("unexpected DELETE path %q", capturedPath)
	}
}

func TestACL_DeleteLayersACLRule_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "rule not found")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteLayersACLRuleContext(context.Background(), ACLRule{Workspace: "topp", Layer: "states", Operation: ACLOpRead})
	assert.False(t, deleted)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
