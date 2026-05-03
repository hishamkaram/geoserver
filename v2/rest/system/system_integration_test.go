//go:build integration

package system_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
)

func TestSystem_ResetCache_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Reset is fast and idempotent — safe to run repeatedly.
	if err := c.System.ResetCache(ctx); err != nil {
		t.Fatalf("ResetCache: %v", err)
	}
}

func TestSystem_Reload_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Reload is heavier but should complete within the test timeout
	// against the lightly-loaded compose stack. It's idempotent.
	if err := c.System.Reload(ctx); err != nil {
		t.Fatalf("Reload: %v", err)
	}
}
