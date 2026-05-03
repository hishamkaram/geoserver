package geoserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hishamkaram/geoserver/v2/internal/transport"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
	"github.com/hishamkaram/geoserver/v2/rest/coveragestores"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/layergroups"
	"github.com/hishamkaram/geoserver/v2/rest/layers"
	"github.com/hishamkaram/geoserver/v2/rest/styles"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

const (
	defaultUserAgent = "geoserver-go/v2"
	defaultTimeout   = 30 * time.Second
)

// Client is the v2 GeoServer REST client. All fields are private; the
// client is configured at construction via [Option] values and is
// immutable thereafter. Concurrent use across goroutines is safe.
//
// Resource methods live on sub-clients accessed via the public fields:
//
//	Workspaces     — workspace CRUD
//	Datastores     — datastore CRUD (workspace-scoped via InWorkspace)
//	FeatureTypes   — feature-type CRUD (workspace+datastore-scoped via InWorkspace().InDatastore())
//	CoverageStores — coverage-store CRUD (workspace-scoped via InWorkspace)
//	Coverages      — coverage CRUD (workspace+coverage-store-scoped via InWorkspace().InCoverageStore())
//	Layers         — layer CRUD (workspace-scoped via InWorkspace)
//	LayerGroups    — layer-group CRUD (workspace-scoped via InWorkspace)
//	Styles         — style metadata + SLD body upload (global by default; .InWorkspace(ws) for workspace-scoped)
//	(more sub-clients as resources port; see ROADMAP.md)
type Client struct {
	core *clientCore

	// Workspaces is the entry point for workspace operations.
	Workspaces *workspaces.Client

	// Datastores is the entry point for datastore operations. Datastore
	// operations are workspace-scoped — see [datastores.Client.InWorkspace].
	Datastores *datastores.Client

	// FeatureTypes is the entry point for feature-type operations.
	// Feature-type operations are 2-level scoped — see
	// [featuretypes.Client.InWorkspace] and [featuretypes.WorkspaceClient.InDatastore].
	FeatureTypes *featuretypes.Client

	// CoverageStores is the entry point for coverage-store operations.
	// Coverage-store operations are workspace-scoped — see
	// [coveragestores.Client.InWorkspace].
	CoverageStores *coveragestores.Client

	// Coverages is the entry point for coverage operations. Coverage
	// operations are 2-level scoped — see [coverages.Client.InWorkspace]
	// and [coverages.WorkspaceClient.InCoverageStore].
	Coverages *coverages.Client

	// Layers is the entry point for layer operations. Layer operations
	// are workspace-scoped — see [layers.Client.InWorkspace].
	Layers *layers.Client

	// LayerGroups is the entry point for layer-group operations.
	// Layer-group operations are workspace-scoped — see
	// [layergroups.Client.InWorkspace].
	LayerGroups *layergroups.Client

	// Styles is the entry point for style operations. The client
	// operates against the global /rest/styles endpoint by default;
	// use [styles.Client.InWorkspace] for a workspace-scoped client.
	Styles *styles.Client
}

// clientCore is the plumbing shared with every sub-client. Sub-clients
// receive a pointer to this and call Do() to issue requests.
type clientCore struct {
	baseURL    string // ends with "/"
	httpClient *http.Client
	logger     *slog.Logger
}

// New constructs an immutable [*Client] for the GeoServer instance at
// serverURL. serverURL must be a valid http:// or https:// URL, with or
// without a trailing slash. Apply options for HTTP client, auth,
// timeout, logging, headers.
//
// Returns an error if any option is invalid or serverURL fails to
// parse — misconfiguration surfaces immediately rather than at
// first-call time.
func New(serverURL string, opts ...Option) (*Client, error) {
	if serverURL == "" {
		return nil, errors.New("geoserver: empty server URL")
	}
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("geoserver: parse server URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("geoserver: server URL scheme must be http or https, got %q", parsed.Scheme)
	}

	cfg := &clientConfig{
		timeout:   defaultTimeout,
		userAgent: defaultUserAgent,
		logger:    slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		if optErr := opt(cfg); optErr != nil {
			return nil, optErr
		}
	}

	httpClient := buildHTTPClient(cfg)

	base := serverURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	core := &clientCore{
		baseURL:    base,
		httpClient: httpClient,
		logger:     cfg.logger,
	}

	c := &Client{core: core}
	adapter := coreAdapter{core: core}
	c.Workspaces = workspaces.New(adapter)
	c.Datastores = datastores.New(adapter)
	c.FeatureTypes = featuretypes.New(adapter)
	c.CoverageStores = coveragestores.New(adapter)
	c.Coverages = coverages.New(adapter)
	c.Layers = layers.New(adapter)
	c.LayerGroups = layergroups.New(adapter)
	c.Styles = styles.New(adapter)
	return c, nil
}

// buildHTTPClient resolves the user's transport options into a single
// *http.Client whose Transport stack is:
//
//	HeaderRoundTripper(user-agent + extra headers) →
//	    AuthRoundTripper(basic | bearer | none) →
//	        cfg.transport or cfg.httpClient.Transport or http.DefaultTransport
//
// If cfg.httpClient is supplied, its Transport is the base and Timeout
// carries through. Otherwise a fresh client is created with cfg.timeout.
func buildHTTPClient(cfg *clientConfig) *http.Client {
	var base http.RoundTripper
	switch {
	case cfg.transport != nil:
		base = cfg.transport
	case cfg.httpClient != nil && cfg.httpClient.Transport != nil:
		base = cfg.httpClient.Transport
	default:
		base = http.DefaultTransport
	}

	// Auth layer (innermost on the request side, outermost on response).
	authed := &transport.AuthRoundTripper{
		Apply: applyAuth(cfg.auth),
		Base:  base,
	}

	// Default headers layer (User-Agent + WithHeader entries).
	headers := http.Header{}
	if cfg.userAgent != "" {
		headers.Set("User-Agent", cfg.userAgent)
	}
	for k, vs := range cfg.defaultHeader {
		for _, v := range vs {
			headers.Add(k, v)
		}
	}
	headed := &transport.HeaderRoundTripper{
		Headers: headers,
		Base:    authed,
	}

	if cfg.httpClient != nil {
		c := *cfg.httpClient
		c.Transport = headed
		if c.Timeout == 0 {
			c.Timeout = cfg.timeout
		}
		return &c
	}
	return &http.Client{Transport: headed, Timeout: cfg.timeout}
}

// applyAuth returns the func attached to the AuthRoundTripper, or nil
// if no auth was configured.
func applyAuth(auth authCredentials) func(*http.Request) {
	switch auth.kind {
	case authBasic:
		username, password := auth.username, auth.password
		return func(r *http.Request) {
			r.SetBasicAuth(username, password)
		}
	case authBearer:
		token := auth.bearer
		return func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
		}
	case authNone:
		return nil
	default:
		return nil
	}
}

// coreAdapter exposes [*clientCore] via the per-resource Core interfaces
// (e.g., [workspaces.Core], [datastores.Core], [featuretypes.Core])
// without the resource subpackages having to import the root package —
// that would create an import cycle since the root imports each
// rest/<resource>.
type coreAdapter struct {
	core *clientCore
}

// URL builds a fully-qualified URL by joining segments onto the
// configured base URL.
func (a coreAdapter) URL(parts ...string) (string, error) {
	return transport.BuildURL(a.core.baseURL, parts)
}

// Do issues a request, decoding the JSON response into out (if non-nil).
// On non-2xx responses, returns a *APIError wrapping the transport-layer
// error. On transport failure, returns the wrapped transport error.
func (a coreAdapter) Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error {
	_, err := transport.DoJSON(ctx, a.core.httpClient, a.core.logger, op, transport.Request{
		Method: method,
		URL:    requestURL,
		Body:   body,
		Query:  query,
	}, out)
	if err == nil {
		return nil
	}
	var tErr *transport.Error
	if errors.As(err, &tErr) {
		return newAPIError(tErr.Op, tErr.Method, tErr.URL, tErr.StatusCode, tErr.Body)
	}
	return err
}

// DoStream issues a request whose response is a stream (e.g., an SLD
// download). The caller owns the returned ReadCloser and must close it.
// Reserved for future resource ports; not used by Workspaces.
func (a coreAdapter) DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error) {
	// Placeholder; expand when the first streaming resource ports.
	return nil, 0, errors.New("geoserver: DoStream not yet implemented")
}

// DoRaw issues a request with an arbitrary-Reader body and explicit
// Content-Type / Accept. Used by sub-clients that need to send non-JSON
// payloads (SLD XML, shapefile zip, GeoTIFF) while keeping JSON-style
// status-to-sentinel error mapping on the response.
//
// If body is nil the request is sent with no payload. If contentType
// is empty "application/octet-stream" is used. If accept is empty
// "application/json" is used.
func (a coreAdapter) DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error {
	_, err := transport.DoRaw(ctx, a.core.httpClient, a.core.logger, op, method, requestURL, body, contentType, accept, query, nil)
	if err == nil {
		return nil
	}
	var tErr *transport.Error
	if errors.As(err, &tErr) {
		return newAPIError(tErr.Op, tErr.Method, tErr.URL, tErr.StatusCode, tErr.Body)
	}
	return err
}
