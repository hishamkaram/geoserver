//go:build integration

package settings_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
)

// Integration test for global settings is read-only. Modifying the
// global settings document on a live integration stack is risky (the
// settings persist across tests and can affect later runs); a write
// path test would need a save/restore harness, which we don't justify
// here. The unit tests cover the Update wire shape directly.
func TestSettings_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.Settings.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("nil settings")
	}
	// GeoServer always ships a non-zero default Charset and at least
	// the JAI block.
	if got.Global.Settings == nil {
		t.Fatalf("Global.Settings nil; full settings = %+v", got)
	}
	if got.Global.Settings.Charset == "" {
		t.Errorf("Charset is empty; expected something like UTF-8: %+v", got.Global.Settings)
	}
}
