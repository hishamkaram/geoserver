package geoserver

import (
	"log/slog"
	"net/http"
	"time"
)

// Option configures a [GeoServer] constructed via [New].
//
// Options are applied in order; later options override earlier ones.
type Option func(*GeoServer)

// New constructs a [*GeoServer] catalog instance for the given GeoServer base
// URL and basic-auth credentials, applying any [Option]s.
//
// Defaults when no options are supplied:
//   - HTTP client: &http.Client{Timeout: 30 * time.Second}
//   - Logger: writes Info-and-above to stderr in slog text format
//   - User-Agent: not set (Go's default)
//
// New is the v1.1+ entry point. The legacy [GetCatalog] is preserved as a
// deprecated wrapper.
func New(serverURL, username, password string, opts ...Option) *GeoServer {
	g := &GeoServer{
		ServerURL:  serverURL,
		Username:   username,
		Password:   password,
		HttpClient: &http.Client{Timeout: defaultHTTPTimeout},
		logger:     GetLogger(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}
	return g
}

// WithHTTPClient sets the *http.Client used for every REST call. The client's
// own Timeout takes precedence over the package default.
//
// Useful for plugging in instrumented transports (OpenTelemetry, custom auth
// RoundTrippers, retryablehttp, etc.).
func WithHTTPClient(c *http.Client) Option {
	return func(g *GeoServer) {
		if c != nil {
			g.HttpClient = c
		}
	}
}

// WithTimeout sets the Timeout on the underlying http.Client. If the client
// was previously customized via [WithHTTPClient], its Timeout is overwritten.
// A non-positive duration disables the request-level timeout.
func WithTimeout(d time.Duration) Option {
	return func(g *GeoServer) {
		if g.HttpClient == nil {
			g.HttpClient = &http.Client{}
		}
		g.HttpClient.Timeout = d
	}
}

// WithLogger configures the library's logger from an [slog.Handler]. Pass nil
// to silence the library entirely (logs are dropped).
//
// Library log levels: Debug for HTTP request/response details, Warn for
// retry-exhausted, Error for protocol failures and decode errors. The default
// (when [WithLogger] is not used) writes Info-and-above to stderr.
func WithLogger(h slog.Handler) Option {
	return func(g *GeoServer) {
		g.logger = loggerFromHandler(h)
	}
}

// WithUserAgent sets a User-Agent header on every outgoing request via a
// transport wrapper around the configured http.Client. Calling [WithUserAgent]
// after [WithHTTPClient] preserves the caller's transport and layers the UA
// header on top.
func WithUserAgent(ua string) Option {
	return func(g *GeoServer) {
		if ua == "" || g.HttpClient == nil {
			return
		}
		base := g.HttpClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		g.HttpClient.Transport = &userAgentTransport{rt: base, ua: ua}
	}
}

// WithBasicAuth overrides the basic-auth credentials on the client. Useful
// when chaining options (e.g. constructing with empty creds and supplying
// them via a later option).
func WithBasicAuth(user, pass string) Option {
	return func(g *GeoServer) {
		g.Username = user
		g.Password = pass
	}
}

// userAgentTransport sets the User-Agent header on every request and delegates
// to an underlying RoundTripper.
type userAgentTransport struct {
	rt http.RoundTripper
	ua string
}

// RoundTrip implements [http.RoundTripper], setting User-Agent on the request.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.ua != "" && req.Header.Get("User-Agent") == "" {
		// Clone first so we don't mutate the caller's request.
		clone := req.Clone(req.Context())
		clone.Header.Set("User-Agent", t.ua)
		return t.rt.RoundTrip(clone)
	}
	return t.rt.RoundTrip(req)
}

// Default slog level used by [GetLogger] / [WithLogger].
//
// Library guidance for log levels:
//   - Debug: per-request URL + status + duration (verbose; opt in via Handler)
//   - Info:  startup-shape events (none in v1.x today)
//   - Warn:  recoverable failures, retries (none implemented yet)
//   - Error: protocol failures, decode errors, transport errors
var _ = slog.LevelInfo
