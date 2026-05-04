//go:build integration

package security_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/security"
)

// uniqueAuthName returns a name unique to this test run; security
// CRUD endpoints don't always tolerate concurrent reuse and the
// test stack persists state across runs without volume reset.
func uniqueAuthName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// ---- AuthProviders ----

func TestSecurity_AuthProviders_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	list, err := c.Security.AuthProviders.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Default GeoServer installs ship at minimum the "default"
	// UsernamePassword provider.
	found := false
	for _, p := range list {
		if p.Name == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected default provider in list, got %+v", list)
	}
}

func TestSecurity_AuthProviders_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	name := uniqueAuthName("v2_it_provider")
	p := &security.AuthProvider{
		Name:                 name,
		ClassName:            "org.geoserver.security.auth.UsernamePasswordAuthenticationProvider",
		UserGroupServiceName: "default",
	}

	t.Cleanup(func() {
		_ = c.Security.AuthProviders.Delete(ctx, name)
	})

	if err := c.Security.AuthProviders.Create(ctx, p, security.CreateAuthProviderOptions{}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := c.Security.AuthProviders.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != name {
		t.Errorf("Get Name = %q, want %q", got.Name, name)
	}
	if got.ClassName != p.ClassName {
		t.Errorf("Get ClassName = %q", got.ClassName)
	}

	if err := c.Security.AuthProviders.Delete(ctx, name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Security.AuthProviders.Get(ctx, name); !errors.Is(err, geoserver.ErrNotFound) {
		t.Errorf("Get after Delete: expected ErrNotFound, got %v", err)
	}
}

func TestSecurity_AuthProviders_SetOrder_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Save current order, restore on cleanup.
	originalList, err := c.Security.AuthProviders.List(ctx)
	if err != nil {
		t.Fatalf("List initial: %v", err)
	}
	originalOrder := make([]string, 0, len(originalList))
	for _, p := range originalList {
		originalOrder = append(originalOrder, p.Name)
	}
	t.Cleanup(func() {
		if len(originalOrder) > 0 {
			_ = c.Security.AuthProviders.SetOrder(ctx, originalOrder)
		}
	})

	// Setting order to ["default"] is a no-op-ish action that
	// exercises the endpoint without changing semantics on a vanilla
	// install.
	if err := c.Security.AuthProviders.SetOrder(ctx, []string{"default"}); err != nil {
		t.Fatalf("SetOrder: %v", err)
	}
}

// ---- AuthFilters ----

func TestSecurity_AuthFilters_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	list, err := c.Security.AuthFilters.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	wantNames := map[string]bool{"anonymous": false, "basic": false}
	for _, ref := range list {
		if _, ok := wantNames[ref.Name]; ok {
			wantNames[ref.Name] = true
		}
	}
	for n, found := range wantNames {
		if !found {
			t.Errorf("expected filter %q in list, got %+v", n, list)
		}
	}
}

func TestSecurity_AuthFilters_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	f, err := c.Security.AuthFilters.Get(ctx, "anonymous")
	if err != nil {
		t.Fatalf("Get anonymous: %v", err)
	}
	if f.Name != "anonymous" {
		t.Errorf("Name = %q", f.Name)
	}
	// Class should be the GeoServer anonymous filter.
	if f.ClassName == "" {
		t.Errorf("ClassName empty in %+v", f)
	}
}

func TestSecurity_AuthFilters_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	_, err := c.Security.AuthFilters.Get(ctx, uniqueAuthName("v2_it_definitely_not_a_filter"))
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ---- FilterChains ----

func TestSecurity_FilterChains_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	list, err := c.Security.FilterChains.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Default GeoServer ships with "web", "rest", "default" chains.
	wantNames := map[string]bool{"web": false, "rest": false, "default": false}
	for _, fc := range list {
		if _, ok := wantNames[fc.Name]; ok {
			wantNames[fc.Name] = true
		}
	}
	for n, found := range wantNames {
		if !found {
			t.Errorf("expected chain %q in list, got chains: %+v", n, chainNames(list))
		}
	}
}

func TestSecurity_FilterChains_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	fc, err := c.Security.FilterChains.Get(ctx, "web")
	if err != nil {
		t.Fatalf("Get web: %v", err)
	}
	if fc.Name != "web" {
		t.Errorf("Name = %q", fc.Name)
	}
	if fc.Path == "" || fc.ClassName == "" {
		t.Errorf("Path/ClassName empty in %+v", fc)
	}
	if len(fc.Filters) == 0 {
		t.Errorf("expected at least one filter in 'web' chain, got %+v", fc)
	}
}

func TestSecurity_FilterChains_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	_, err := c.Security.FilterChains.Get(ctx, uniqueAuthName("v2_it_definitely_not_a_chain"))
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func chainNames(list []security.FilterChain) []string {
	out := make([]string, len(list))
	for i, fc := range list {
		out[i] = fc.Name
	}
	return out
}
