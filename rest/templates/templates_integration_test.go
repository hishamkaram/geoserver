//go:build integration

package templates_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
)

// Templates ride on /workspaces/{ws}/... etc., so the test scope
// uses the always-present "topp" workspace and the always-present
// "nurc/mosaic/mosaic" coverage. Naming with a nanosecond timestamp
// avoids collisions across runs (the test suite tries to clean up
// but skipped/aborted runs may leave templates behind).
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func TestTemplates_Global_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	name := uniqueName("v2_it_global")
	body := "Hello from " + name + "\n"

	t.Cleanup(func() { _ = c.Templates.Delete(ctx, name) })

	if err := c.Templates.PutString(ctx, name, body); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := c.Templates.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != body {
		t.Errorf("Get = %q, want %q", got, body)
	}

	// List should include the new template.
	list, err := c.Templates.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := name + ".ftl"
	found := false
	for _, ref := range list {
		if ref.Name == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("template %q not in List", want)
	}

	if err := c.Templates.Delete(ctx, name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Templates.Get(ctx, name); !errors.Is(err, geoserver.ErrNotFound) {
		t.Errorf("Get after Delete: expected ErrNotFound, got %v", err)
	}
}

func TestTemplates_Workspace_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	scope := c.Templates.InWorkspace("topp")
	name := uniqueName("v2_it_ws")
	body := "ws scope " + name

	t.Cleanup(func() { _ = scope.Delete(ctx, name) })

	if err := scope.PutString(ctx, name, body); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := scope.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != body {
		t.Errorf("Get = %q", got)
	}
	if err := scope.Delete(ctx, name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestTemplates_Coverage_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	scope := c.Templates.InWorkspace("nurc").
		InCoverageStore("mosaic").
		InCoverage("mosaic")
	name := uniqueName("v2_it_cov")
	body := "coverage scope " + name

	t.Cleanup(func() { _ = scope.Delete(ctx, name) })

	if err := scope.PutString(ctx, name, body); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if got, err := scope.Get(ctx, name); err != nil {
		t.Fatalf("Get: %v", err)
	} else if got != body {
		t.Errorf("Get = %q", got)
	}
	// Confirm the new template is listed at the coverage scope.
	list, err := scope.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := name + ".ftl"
	found := false
	for _, ref := range list {
		if ref.Name == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("template %q not listed; saw %v", want, list)
	}
}

func TestTemplates_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)
	_, err := c.Templates.Get(ctx, uniqueName("v2_it_definitely_not_a_template"))
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTemplates_AcceptsNameWithFTLSuffix_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)
	name := uniqueName("v2_it_suffix") + ".ftl"

	t.Cleanup(func() { _ = c.Templates.Delete(ctx, name) })
	if err := c.Templates.PutString(ctx, name, "x"); err != nil {
		t.Fatalf("Put with .ftl suffix: %v", err)
	}
	if got, err := c.Templates.Get(ctx, strings.TrimSuffix(name, ".ftl")); err != nil {
		t.Fatalf("Get without .ftl suffix: %v", err)
	} else if got != "x" {
		t.Errorf("body = %q", got)
	}
}
