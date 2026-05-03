package settings

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
}

// Client is the v2 settings sub-client. There is no list / create —
// the global settings document is a singleton.
//
//	settings, _ := c.Settings.Get(ctx)
//	settings.Global.Settings.Charset = "UTF-8"
//	_ = c.Settings.Update(ctx, settings)
type Client struct {
	core Core
}

// New constructs the sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// Get fetches the global settings document.
func (c *Client) Get(ctx context.Context) (*Settings, error) {
	const op = "Settings.Get"
	u, err := c.core.URL("rest", "settings")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var s Settings
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Update performs a partial update of the global settings document.
//
// Fields with their zero value (e.g., bool=false, string="") and
// `omitempty` JSON tags are dropped from the wire form. To set a bool
// to false explicitly, fetch the current settings, mutate the field,
// and put the document back — the Get-modify-Put pattern preserves
// the existing zero-valued fields you didn't intend to change.
func (c *Client) Update(ctx context.Context, settings *Settings) error {
	const op = "Settings.Update"
	if settings == nil {
		return errors.New(op + ": nil settings")
	}
	u, err := c.core.URL("rest", "settings")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, settings, nil, nil)
}
