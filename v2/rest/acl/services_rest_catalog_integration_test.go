//go:build integration

package acl_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
)

func TestACL_Services_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// GeoServer's services-ACL validator rejects rules where the
	// service is "*" with a non-"*" operation (error message:
	// "when namespace is * then also layer must be *"), and rejects
	// rules referencing operations that don't exist in any real
	// service. Pin to (wfs, GetCapabilities) — a canonical operation
	// that's vanishingly unlikely to have a custom rule already on a
	// fresh GeoServer install. Uniqueness is provided by the role
	// name; the rule itself is removed on cleanup.
	roleName := testenv.UniqueName(t, "ROLE")

	rule := acl.ServiceRule{
		Service:   "wfs",
		Operation: "GetCapabilities",
		Roles:     []string{roleName},
	}

	t.Cleanup(func() {
		_ = c.ACL.Services().Delete(ctx, rule)
	})

	if err := c.ACL.Services().Add(ctx, rule); err != nil {
		t.Fatalf("Add service rule: %v", err)
	}

	rules, err := c.ACL.Services().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List service rules: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.Service == "wfs" && r.Operation == "GetCapabilities" {
			if len(r.Roles) == 1 && r.Roles[0] == roleName {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("rule for wfs.GetCapabilities with role %q not found in List", roleName)
	}

	// Update: change the role list.
	updatedRoleName := testenv.UniqueName(t, "ROLE")
	updated := rule
	updated.Roles = []string{roleName, updatedRoleName}
	if err := c.ACL.Services().Update(ctx, updated); err != nil {
		t.Fatalf("Update service rule: %v", err)
	}

	// Verify the update landed.
	rules, err = c.ACL.Services().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List after Update: %v", err)
	}
	gotRoles := []string(nil)
	for _, r := range rules {
		if r.Service == "wfs" && r.Operation == "GetCapabilities" {
			gotRoles = r.Roles
			break
		}
	}
	if len(gotRoles) != 2 {
		t.Fatalf("expected 2 roles after Update, got %v", gotRoles)
	}

	if err := c.ACL.Services().Delete(ctx, rule); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Confirm removal.
	rules, err = c.ACL.Services().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	for _, r := range rules {
		if r.Service == "wfs" && r.Operation == "GetCapabilities" {
			t.Fatalf("rule still present after Delete: %+v", r)
		}
	}
}

func TestACL_REST_AddListUpdate_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// REST ACL rule keys carry "/" and ":" in their URL-path form,
	// which GeoServer's HTTP firewall rejects by default — so the
	// DELETE /rest/security/acl/rest/{rule} endpoint is effectively
	// non-functional on a default install. See [RESTClient] doc for
	// the full caveat. This test exercises the Add/List/Update path
	// (which all hit the list endpoint and are unaffected); cleanup
	// best-effort tries Delete and tolerates failure.
	//
	// Pattern suffix uses a nanosecond timestamp because rules
	// accumulate across runs (Delete is broken on default GeoServer);
	// testenv.UniqueName's per-process counter would re-collide on
	// each run.
	patternSuffix := fmt.Sprintf("fake_%d", time.Now().UnixNano())
	roleName := testenv.UniqueName(t, "ROLE")

	rule := acl.RESTRule{
		Pattern: "/rest/" + patternSuffix + "/**",
		Methods: []string{"GET"},
		Roles:   []string{roleName},
	}

	t.Cleanup(func() {
		// Best-effort Delete. If GeoServer's firewall rejects this
		// path (the common case on default installs), the rule will
		// remain on the server until manually cleaned.
		_ = c.ACL.REST().Delete(ctx, rule)
	})

	if err := c.ACL.REST().Add(ctx, rule); err != nil {
		t.Fatalf("Add REST rule: %v", err)
	}

	rules, err := c.ACL.REST().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List REST rules: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.Pattern == rule.Pattern && len(r.Methods) == 1 && r.Methods[0] == "GET" {
			if len(r.Roles) == 1 && r.Roles[0] == roleName {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("rule for %s:GET not found in List", rule.Pattern)
	}

	// Update: change the role list.
	updatedRoleName := testenv.UniqueName(t, "ROLE")
	updated := rule
	updated.Roles = []string{roleName, updatedRoleName}
	if err := c.ACL.REST().Update(ctx, updated); err != nil {
		t.Fatalf("Update REST rule: %v", err)
	}

	// Verify Update landed.
	rules, err = c.ACL.REST().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List after Update: %v", err)
	}
	gotRoles := []string(nil)
	for _, r := range rules {
		if r.Pattern == rule.Pattern && len(r.Methods) == 1 && r.Methods[0] == "GET" {
			gotRoles = r.Roles
			break
		}
	}
	if len(gotRoles) != 2 {
		t.Errorf("expected 2 roles after Update, got %v", gotRoles)
	}
}

func TestACL_REST_Delete_Integration(t *testing.T) {
	// Standalone Delete test — skipped by default because REST ACL
	// DELETE requires Spring Security configuration that is not
	// exposed via env vars on a default GeoServer install. See
	// [acl.RESTClient] doc for the full explanation. Set the env
	// var GEOSERVER_REST_ACL_DELETE_WORKS=1 to run it on a
	// custom-configured server.
	if os.Getenv("GEOSERVER_REST_ACL_DELETE_WORKS") != "1" {
		t.Skip("REST ACL DELETE requires Spring Security firewall config not present on default GeoServer; see acl.RESTClient godoc for details")
	}

	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	patternSuffix := fmt.Sprintf("fake_%d", time.Now().UnixNano())
	roleName := testenv.UniqueName(t, "ROLE")

	rule := acl.RESTRule{
		Pattern: "/rest/" + patternSuffix + "/**",
		Methods: []string{"GET"},
		Roles:   []string{roleName},
	}
	t.Cleanup(func() {
		_ = c.ACL.REST().Delete(ctx, rule)
	})
	if err := c.ACL.REST().Add(ctx, rule); err != nil {
		t.Fatalf("Add REST rule: %v", err)
	}
	if err := c.ACL.REST().Delete(ctx, rule); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	rules, err := c.ACL.REST().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	for _, r := range rules {
		if r.Pattern == rule.Pattern && len(r.Methods) == 1 && r.Methods[0] == "GET" {
			t.Fatalf("rule still present after Delete: %+v", r)
		}
	}
}

func TestACL_Catalog_GetUpdate_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	original, err := c.ACL.Catalog().Get(ctx)
	if err != nil {
		t.Fatalf("Get catalog mode: %v", err)
	}
	if original != acl.CatalogModeHide && original != acl.CatalogModeMixed && original != acl.CatalogModeChallenge {
		t.Fatalf("unexpected initial catalog mode %q", original)
	}

	// Restore on cleanup so we don't leave the server in an altered
	// state (subsequent tests may rely on the default mode).
	t.Cleanup(func() {
		_ = c.ACL.Catalog().Update(ctx, original)
	})

	// Toggle to a non-original mode.
	target := acl.CatalogModeMixed
	if original == acl.CatalogModeMixed {
		target = acl.CatalogModeHide
	}
	if err := c.ACL.Catalog().Update(ctx, target); err != nil {
		t.Fatalf("Update catalog mode to %q: %v", target, err)
	}

	got, err := c.ACL.Catalog().Get(ctx)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got != target {
		t.Errorf("after Update mode = %q, want %q", got, target)
	}
}

func TestACL_Catalog_Update_InvalidMode_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	err := c.ACL.Catalog().Update(ctx, acl.CatalogMode("BOGUS"))
	if err == nil {
		t.Fatalf("expected error for invalid catalog mode, got nil")
	}
	// GeoServer documents 422 for invalid modes; surface as APIError
	// regardless of which specific 4xx it ends up being.
	var apiErr *geoserver.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
		t.Errorf("expected 4xx status, got %d", apiErr.StatusCode)
	}
}

func TestACL_Catalog_Reload_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	if err := c.ACL.Catalog().Reload(ctx); err != nil {
		t.Fatalf("Reload: %v", err)
	}
}
