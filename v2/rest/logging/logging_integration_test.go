//go:build integration

package logging_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/logging"
)

func TestLogging_GetUpdate_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	original, err := c.Logging.Get(ctx)
	if err != nil {
		t.Fatalf("Get original: %v", err)
	}
	if original.Level == "" {
		t.Errorf("expected non-empty Level, got %+v", original)
	}

	// Restore on cleanup so subsequent tests / dev sessions see the
	// original configuration.
	t.Cleanup(func() {
		_ = c.Logging.Update(ctx, original)
	})

	// Toggle to a non-original level. Default GeoServer ships
	// DEFAULT_LOGGING / VERBOSE_LOGGING / QUIET_LOGGING /
	// PRODUCTION_LOGGING / GEOSERVER_DEVELOPER_LOGGING /
	// GEOTOOLS_DEVELOPER_LOGGING / TEST_LOGGING in logs/.
	target := "VERBOSE_LOGGING"
	if original.Level == "VERBOSE_LOGGING" {
		target = "DEFAULT_LOGGING"
	}
	cfg := &logging.Config{
		Level:         target,
		StdOutLogging: original.StdOutLogging,
	}
	if err := c.Logging.Update(ctx, cfg); err != nil {
		t.Fatalf("Update to %q: %v", target, err)
	}

	got, err := c.Logging.Get(ctx)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Level != target {
		t.Errorf("after Update Level = %q, want %q", got.Level, target)
	}
}
