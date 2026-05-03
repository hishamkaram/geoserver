package geoserver

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Option configures a [*Client] at construction. Options are applied in
// order; later options override earlier ones for the same field.
type Option func(*clientConfig) error

// clientConfig is the internal accumulator of options. Resolved into a
// *Client by [New].
type clientConfig struct {
	httpClient    *http.Client
	timeout       time.Duration
	transport     http.RoundTripper
	logger        *slog.Logger
	userAgent     string
	auth          authCredentials
	defaultHeader http.Header
}

// authCredentials holds the resolved auth strategy. Mutually exclusive:
// at most one of basic / bearer is set. Empty == no auth header attached.
type authCredentials struct {
	kind     authKind
	username string
	password string
	bearer   string
}

type authKind int

const (
	authNone authKind = iota
	authBasic
	authBearer
)

// WithHTTPClient supplies a custom *http.Client. The client's Transport
// (if any) is wrapped — auth and user-agent layers are applied on top.
// Mutually exclusive with [WithTransport]; later option wins.
func WithHTTPClient(c *http.Client) Option {
	return func(cfg *clientConfig) error {
		if c == nil {
			return errors.New("geoserver: WithHTTPClient: client is nil")
		}
		cfg.httpClient = c
		return nil
	}
}

// WithTransport supplies the base [http.RoundTripper] for the client's
// HTTP transport. Auth and user-agent layers wrap this. Mutually
// exclusive with [WithHTTPClient]; later option wins.
func WithTransport(rt http.RoundTripper) Option {
	return func(cfg *clientConfig) error {
		if rt == nil {
			return errors.New("geoserver: WithTransport: transport is nil")
		}
		cfg.transport = rt
		return nil
	}
}

// WithTimeout sets the *http.Client's Timeout. Zero means no timeout
// (rely on context deadlines instead). Default: 30 seconds.
func WithTimeout(d time.Duration) Option {
	return func(cfg *clientConfig) error {
		if d < 0 {
			return errors.New("geoserver: WithTimeout: negative duration")
		}
		cfg.timeout = d
		return nil
	}
}

// WithLogger sets the *slog.Logger the client uses for HTTP-level
// logging. Default: a discard handler (silent).
func WithLogger(l *slog.Logger) Option {
	return func(cfg *clientConfig) error {
		if l == nil {
			return errors.New("geoserver: WithLogger: logger is nil; pass slog.New(slog.DiscardHandler) to silence")
		}
		cfg.logger = l
		return nil
	}
}

// WithUserAgent sets the User-Agent header sent on every request.
// Default: "geoserver-go/v2".
func WithUserAgent(ua string) Option {
	return func(cfg *clientConfig) error {
		if ua == "" {
			return errors.New("geoserver: WithUserAgent: empty user-agent")
		}
		cfg.userAgent = ua
		return nil
	}
}

// WithBasicAuth attaches HTTP basic-auth headers to every request.
// Mutually exclusive with [WithBearerToken]; later option wins.
func WithBasicAuth(username, password string) Option {
	return func(cfg *clientConfig) error {
		if username == "" {
			return errors.New("geoserver: WithBasicAuth: empty username")
		}
		cfg.auth = authCredentials{kind: authBasic, username: username, password: password}
		return nil
	}
}

// WithBearerToken attaches a bearer token to every request.
// Mutually exclusive with [WithBasicAuth]; later option wins.
func WithBearerToken(token string) Option {
	return func(cfg *clientConfig) error {
		if token == "" {
			return errors.New("geoserver: WithBearerToken: empty token")
		}
		cfg.auth = authCredentials{kind: authBearer, bearer: token}
		return nil
	}
}

// WithHeader adds a default header sent on every request. Multiple calls
// accumulate. Authoritative for headers set before request-level
// overrides apply.
func WithHeader(key, value string) Option {
	return func(cfg *clientConfig) error {
		if key == "" {
			return errors.New("geoserver: WithHeader: empty key")
		}
		if cfg.defaultHeader == nil {
			cfg.defaultHeader = http.Header{}
		}
		cfg.defaultHeader.Add(key, value)
		return nil
	}
}
