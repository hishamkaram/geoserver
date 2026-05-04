//go:build integration

package security_test

import (
	"os"
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
)

// TestMasterPassword_GetUpdate_RoundTrip exercises GET + PUT against
// the live stack. GeoServer's master-password endpoint refuses a
// same-value update (422 "Cannot change master password"), so the
// test rotates to a strong throwaway value and reverts to the
// original in t.Cleanup so subsequent tests / dev sessions are
// unaffected.
func TestMasterPassword_GetUpdate_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	original, err := c.Security.MasterPassword.Get(ctx)
	if err != nil {
		t.Fatalf("Get original: %v", err)
	}
	if original == "" {
		t.Fatal("expected non-empty master password, got empty string")
	}

	const rotated = "GeoServer-MASTER-Test-Rotation-2026!"

	// Belt-and-braces revert in case the test panics mid-way.
	t.Cleanup(func() {
		_ = c.Security.MasterPassword.Update(ctx, rotated, original)
	})

	if err := c.Security.MasterPassword.Update(ctx, original, rotated); err != nil {
		t.Fatalf("Update to rotated: %v", err)
	}

	got, err := c.Security.MasterPassword.Get(ctx)
	if err != nil {
		t.Fatalf("Get after rotation: %v", err)
	}
	if got != rotated {
		t.Errorf("after rotation got %q, want %q", got, rotated)
	}

	// Explicit revert (also catches errors the Cleanup would swallow).
	if err := c.Security.MasterPassword.Update(ctx, rotated, original); err != nil {
		t.Fatalf("revert: %v", err)
	}
}

// TestSelfPassword_Change_Idempotent calls Change with the user's
// current password (the value testenv.NewClient is already
// authenticating with). If the endpoint applies the change, it's a
// no-op; if it returns 200 without applying (observed on some 2.x
// versions), the test still asserts the contract — body shape and
// auth are accepted.
//
// We deliberately do NOT change the password to a different value
// because subsequent integration tests authenticate with the same
// account and would fail.
func TestSelfPassword_Change_Idempotent_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	pass := testenv.DefaultPass
	if env := os.Getenv("GEOSERVER_PASS"); env != "" {
		pass = env
	}
	if err := c.Security.SelfPassword.Change(ctx, pass); err != nil {
		t.Fatalf("Change (idempotent): %v", err)
	}
}
