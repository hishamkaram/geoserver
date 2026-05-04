package security

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// MasterPasswordClient covers the master password singleton at
// /rest/security/masterpw. The master password unlocks GeoServer's
// keystore (used for storing connection-string passwords, JKS aliases,
// etc.); it is distinct from the admin user's login password handled
// by [SelfPasswordClient]. Operations require admin auth.
//
//	pw, _ := c.Security.MasterPassword.Get(ctx)
//	_ = c.Security.MasterPassword.Update(ctx, pw, "new-strong-secret")
type MasterPasswordClient struct {
	core Core
}

// masterPasswordResponse matches the wire shape of GET /rest/security/masterpw:
// `{"oldMasterPassword":"<current value>"}`. The "old" prefix is
// awkward — semantically the GET returns the *current* master
// password — but it matches the upstream API.
type masterPasswordResponse struct {
	OldMasterPassword string `json:"oldMasterPassword"`
}

// masterPasswordChangeRequest is the PUT body. Both fields are
// required by GeoServer; the server validates oldMasterPassword
// against the current value before applying newMasterPassword.
type masterPasswordChangeRequest struct {
	OldMasterPassword string `json:"oldMasterPassword"`
	NewMasterPassword string `json:"newMasterPassword"`
}

// Get returns the current master password as a plain string.
//
// GeoServer exposes the master password via GET (gated by admin auth)
// to support backup / disaster-recovery flows. Treat the returned
// value with the same care as any other secret.
func (c *MasterPasswordClient) Get(ctx context.Context) (string, error) {
	const op = "Security.MasterPassword.Get"
	u, err := c.core.URL("rest", "security", "masterpw")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	var resp masterPasswordResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return "", err
	}
	return resp.OldMasterPassword, nil
}

// Update changes the master password. The caller MUST supply the
// current value as oldMasterPassword; GeoServer rejects the request
// (422 / 401, depending on version) when it doesn't match.
//
// Both arguments are required and must be non-empty.
func (c *MasterPasswordClient) Update(ctx context.Context, oldMasterPassword, newMasterPassword string) error {
	const op = "Security.MasterPassword.Update"
	if oldMasterPassword == "" {
		return errors.New(op + ": empty oldMasterPassword")
	}
	if newMasterPassword == "" {
		return errors.New(op + ": empty newMasterPassword")
	}
	u, err := c.core.URL("rest", "security", "masterpw")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := masterPasswordChangeRequest{
		OldMasterPassword: oldMasterPassword,
		NewMasterPassword: newMasterPassword,
	}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}
