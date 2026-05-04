// Package logging is the v2 sub-client for the GeoServer
// /rest/logging endpoint — singleton configuration for the active
// log4j profile and stdout logging.
//
// The endpoint is small but useful for production debugging:
// changing the log level without bouncing the server is the daily
// driver. Per the upstream API, log location cannot be changed
// through REST in GeoServer 3.0+ (it is read-only there); the SDK
// preserves Location on round-trips for compatibility with older
// 2.x deployments.
package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
}

// Config is the GeoServer logging configuration document.
type Config struct {
	// Level names a log4j profile bundled into the GeoServer data
	// directory's logs/ subdirectory (e.g. "DEFAULT_LOGGING",
	// "VERBOSE_LOGGING", "QUIET_LOGGING", "PRODUCTION_LOGGING",
	// "GEOSERVER_DEVELOPER_LOGGING").
	Level string `json:"level,omitempty"`
	// Location is the on-disk log file path
	// (e.g. "logs/geoserver.log"). Read-only since GeoServer 3.0;
	// PUT bodies that include this field have it ignored.
	Location string `json:"location,omitempty"`
	// StdOutLogging mirrors logging to the GeoServer container's
	// standard output.
	StdOutLogging bool `json:"stdOutLogging,omitempty"`
}

// MarshalJSON wraps Config in GeoServer's `{"logging":{...}}`
// envelope expected by PUT bodies.
func (c Config) MarshalJSON() ([]byte, error) {
	type alias Config
	return json.Marshal(map[string]alias{"logging": alias(c)})
}

// UnmarshalJSON accepts both the wrapped and the flat shape.
func (c *Config) UnmarshalJSON(b []byte) error {
	type alias Config
	var wrapped struct {
		Logging *alias `json:"logging"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && wrapped.Logging != nil {
		*c = Config(*wrapped.Logging)
		return nil
	}
	var flat alias
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*c = Config(flat)
	return nil
}

// Client is the v2 logging sub-client.
type Client struct {
	core Core
}

// New constructs the logging sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// Get returns the current logging configuration.
func (c *Client) Get(ctx context.Context) (*Config, error) {
	const op = "Logging.Get"
	u, err := c.core.URL("rest", "logging")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var cfg Config
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Update replaces the logging configuration.
func (c *Client) Update(ctx context.Context, cfg *Config) error {
	const op = "Logging.Update"
	if cfg == nil {
		return errors.New(op + ": nil config")
	}
	u, err := c.core.URL("rest", "logging")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, cfg, nil, nil)
}
