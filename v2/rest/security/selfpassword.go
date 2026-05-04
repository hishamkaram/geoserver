package security

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// SelfPasswordClient covers PUT /rest/security/self/password — the
// authenticated user's own login password. The endpoint is GET-less
// by design (GeoServer responds 405 "You can not request the
// password!"); this client therefore exposes a single Change method.
//
//	_ = c.Security.SelfPassword.Change(ctx, "new-strong-secret")
//
// To change another user's password, use the user/group-service path
// via [UsersClient] (admin-gated). SelfPassword is intentionally the
// thin "the auth'd user rotates their own password" path.
type SelfPasswordClient struct {
	core Core
}

// selfPasswordChangeRequest matches the PUT body wire shape:
// `{"newPassword":"..."}`. The endpoint authenticates the caller via
// the request's auth header — there is no oldPassword field; the
// auth header itself proves possession.
type selfPasswordChangeRequest struct {
	NewPassword string `json:"newPassword"`
}

// Change rotates the authenticated user's password to newPassword.
//
// Returns an [APIError] wrapping [ErrUnauthorized] if the caller's
// credentials are no longer valid, [ErrBadRequest] if newPassword
// fails the password policy.
func (c *SelfPasswordClient) Change(ctx context.Context, newPassword string) error {
	const op = "Security.SelfPassword.Change"
	if newPassword == "" {
		return errors.New(op + ": empty newPassword")
	}
	u, err := c.core.URL("rest", "security", "self", "password")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := selfPasswordChangeRequest{NewPassword: newPassword}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}
