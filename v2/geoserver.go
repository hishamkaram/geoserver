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
	"github.com/hishamkaram/geoserver/v2/rest/about"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
	"github.com/hishamkaram/geoserver/v2/rest/coveragestores"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/fonts"
	"github.com/hishamkaram/geoserver/v2/rest/gwc"
	"github.com/hishamkaram/geoserver/v2/rest/imports"
	"github.com/hishamkaram/geoserver/v2/rest/layergroups"
	"github.com/hishamkaram/geoserver/v2/rest/layers"
	"github.com/hishamkaram/geoserver/v2/rest/logging"
	"github.com/hishamkaram/geoserver/v2/rest/namespaces"
	"github.com/hishamkaram/geoserver/v2/rest/resources"
	"github.com/hishamkaram/geoserver/v2/rest/security"
	"github.com/hishamkaram/geoserver/v2/rest/services"
	"github.com/hishamkaram/geoserver/v2/rest/settings"
	"github.com/hishamkaram/geoserver/v2/rest/styles"
	"github.com/hishamkaram/geoserver/v2/rest/system"
	"github.com/hishamkaram/geoserver/v2/rest/templates"
	"github.com/hishamkaram/geoserver/v2/rest/urlchecks"
	"github.com/hishamkaram/geoserver/v2/rest/wfstransforms"
	"github.com/hishamkaram/geoserver/v2/rest/wmslayers"
	"github.com/hishamkaram/geoserver/v2/rest/wmsstores"
	"github.com/hishamkaram/geoserver/v2/rest/wmtslayers"
	"github.com/hishamkaram/geoserver/v2/rest/wmtsstores"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"

	"github.com/hishamkaram/geoserver/v2/ows/wcs"
	"github.com/hishamkaram/geoserver/v2/ows/wfs"
	"github.com/hishamkaram/geoserver/v2/ows/wms"
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
//	Namespaces     — namespace CRUD (flat global)
//	Settings       — singleton global settings document (Get / Update)
//	About          — health check + version info (Ping / Version)
//	Security       — users, groups, roles, user-role assignment
//	ACL            — access-control rules (Layers; future: Services, Catalog)
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

	// Namespaces is the entry point for namespace operations.
	// Namespaces are flat under /rest/namespaces.
	Namespaces *namespaces.Client

	// Settings is the entry point for the singleton global-settings
	// document — Get / Update against /rest/settings.
	Settings *settings.Client

	// About is the entry point for server health and version info —
	// Ping (liveness check) and Version (full component versions).
	About *about.Client

	// Security is the entry point for users, groups, roles, and
	// user-role assignment under /rest/security. See
	// [security.Client] for the nested sub-client surface.
	Security *security.Client

	// ACL is the entry point for access-control-list rules under
	// /rest/security/acl. Currently exposes layer ACLs via
	// [acl.Client.Layers]; service-level and catalog-level ACL
	// endpoints can be added in follow-up PRs.
	ACL *acl.Client

	// System is the entry point for server-management operations —
	// Reload (catalog + configuration from disk) and ResetCache
	// (store / raster / schema caches). Both require admin auth.
	System *system.Client

	// WMS is the entry point for WMS service operations — currently
	// GetCapabilities (XML, decoded into [wms.Capabilities]). Use
	// [wms.Client.InWorkspace] for a workspace-scoped capabilities
	// document.
	WMS *wms.Client

	// WFS is the entry point for WFS service operations — currently
	// GetCapabilities (XML, decoded into [wfs.Capabilities]). Use
	// [wfs.Client.InWorkspace] for a workspace-scoped capabilities
	// document.
	WFS *wfs.Client

	// WCS is the entry point for WCS service operations — currently
	// GetCapabilities (XML, decoded into [wcs.Capabilities]). Use
	// [wcs.Client.InWorkspace] for a workspace-scoped capabilities
	// document.
	WCS *wcs.Client

	// Imports is the entry point for the GeoServer Importer
	// extension at /rest/imports — bulk-ingest sessions for batch
	// publishing, migrations, and drop-and-republish workflows.
	// The extension is NOT installed by default; calls to a
	// GeoServer without it return ErrNotFound.
	Imports *imports.Client

	// GWC is the entry point for GeoWebCache REST endpoints —
	// per-layer cache config, seed/reseed/truncate tasks, and
	// disk-quota policy. Lives at /gwc/rest/ (outside the /rest/
	// catalog tree). Reach the typed sub-clients via
	// `c.GWC.Layers()`, `Seed()`, `DiskQuota()`.
	GWC *gwc.Client

	// Services is the entry point for per-service OWS configuration
	// (`/services/{wms,wfs,wcs,wmts}/settings`). Reach the typed
	// per-service clients via `c.Services.WMS()`, `WFS()`, `WCS()`,
	// `WMTS()`. Each exposes `Get`/`Update` for global settings and
	// `.InWorkspace(ws)` for per-workspace overrides (`Get`/`Update`/`Delete`).
	Services *services.Client

	// Resources is the entry point for the GeoServer Resource API at
	// /rest/resource/{path} — generic byte-stream access to files
	// in the GeoServer data directory. Daily-driver methods cover
	// reading file contents, listing directories, uploading new
	// resources (FTL templates, SLD includes, icons), moving /
	// copying files, and recursive deletion.
	Resources *resources.Client

	// Templates is the entry point for FreeMarker (FTL) templates
	// that customize GetFeatureInfo HTML output, WMS HTML
	// capabilities, and other text outputs. Six scope levels are
	// supported — global, per workspace, per datastore, per feature
	// type, per coverage store, per coverage — via fluent
	// `c.Templates.InWorkspace(ws).In...()` chains.
	Templates *templates.Client

	// URLChecks is the entry point for URL External Access Checks
	// at /rest/urlchecks. Allow/deny lists for external URL
	// references in styles, mosaics, and remote stores. Used by
	// SSRF-conscious deployments to constrain which off-server
	// URLs GeoServer is permitted to fetch.
	URLChecks *urlchecks.Client

	// WMSStores is the entry point for cascaded WMS stores —
	// references to remote WMS servers re-published through this
	// GeoServer instance. Workspace-scoped via
	// `c.WMSStores.InWorkspace(ws)`.
	WMSStores *wmsstores.Client

	// WMSLayers is the entry point for cascaded WMS layers —
	// individual remote WMS layers published locally. 2-level
	// scoped: `c.WMSLayers.InWorkspace(ws).InStore(s)` for the
	// canonical CRUD path, `c.WMSLayers.InWorkspace(ws)` for the
	// workspace-wide list / get / delete.
	WMSLayers *wmslayers.Client

	// WMTSStores is the entry point for cascaded WMTS stores.
	// Same scoping pattern as [Client.WMSStores].
	WMTSStores *wmtsstores.Client

	// WMTSLayers is the entry point for cascaded WMTS layers.
	// Same scoping pattern as [Client.WMSLayers].
	WMTSLayers *wmtslayers.Client

	// WFSTransforms is the entry point for XSLT transforms that
	// re-shape WFS GetFeature output (HTML reports, KML, custom XML).
	// The endpoint is part of the gs-xslt-wfs extension; calls
	// against an unequipped GeoServer return ErrNotFound.
	WFSTransforms *wfstransforms.Client

	// Logging is the entry point for the singleton logging
	// configuration document at /rest/logging — adjust the active
	// log4j profile (DEFAULT_LOGGING, VERBOSE_LOGGING, etc.) and
	// the stdout-mirror toggle without bouncing the server.
	Logging *logging.Client

	// Fonts is the entry point for the read-only /rest/fonts
	// endpoint — list of font families the JVM exposes to GeoServer's
	// SLD labelling pipeline. Sanity-check before publishing styles
	// that reference specific fonts; typos would otherwise surface as
	// silent label-rendering fallbacks.
	Fonts *fonts.Client
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
	c.Namespaces = namespaces.New(adapter)
	c.Settings = settings.New(adapter)
	c.About = about.New(adapter)
	c.Security = security.New(adapter)
	c.ACL = acl.New(adapter)
	c.System = system.New(adapter)
	c.WMS = wms.New(adapter)
	c.WFS = wfs.New(adapter)
	c.WCS = wcs.New(adapter)
	c.Services = services.New(adapter)
	c.GWC = gwc.New(adapter)
	c.Imports = imports.New(adapter)
	c.Resources = resources.New(adapter)
	c.Templates = templates.New(adapter)
	c.URLChecks = urlchecks.New(adapter)
	c.WMSStores = wmsstores.New(adapter)
	c.WMSLayers = wmslayers.New(adapter)
	c.WMTSStores = wmtsstores.New(adapter)
	c.WMTSLayers = wmtslayers.New(adapter)
	c.WFSTransforms = wfstransforms.New(adapter)
	c.Logging = logging.New(adapter)
	c.Fonts = fonts.New(adapter)
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

// DoStream issues a request whose response is a streamed body
// (e.g., a Resource API file download whose Content-Type can be
// XML / JSON / image / binary depending on the file). The caller
// owns the returned [io.ReadCloser] and must close it.
//
// On 2xx, returns the open body, the status code, and nil error.
// On non-2xx, drains and closes the body, returns a [*APIError].
// On transport failure, returns the wrapped transport error.
func (a coreAdapter) DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: build request: %w", op, err)
	}
	httpReq.Header.Set("Accept", "*/*")
	if len(query) > 0 {
		q := httpReq.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}
	resp, err := a.core.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %s %s: %w", op, method, requestURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		_ = resp.Body.Close()
		return nil, resp.StatusCode, newAPIError(op, method, requestURL, resp.StatusCode, body)
	}
	return resp.Body, resp.StatusCode, nil
}

// DoXML issues a GET-style request and decodes the response as XML.
// Used by the OWS sub-clients (WMS / WFS / WCS) for GetCapabilities
// and similar XML endpoints. Response body cap is 32 MiB to handle
// large capabilities documents; the JSON 8 KiB cap on [Do] would
// truncate them.
func (a coreAdapter) DoXML(ctx context.Context, op, method, requestURL string, query map[string]string, out any) error {
	_, err := transport.DoXML(ctx, a.core.httpClient, a.core.logger, op, transport.Request{
		Method: method,
		URL:    requestURL,
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

// SynthesizeError manufactures an [*APIError] with the supplied
// status code, suitable for sub-clients that need to surface a
// package sentinel (e.g., [ErrNotFound]) when the wire response is
// semantically an error but technically a 2xx — for example, the
// Resource API returns 200 with type="undefined" for missing paths
// when queried with operation=metadata, and the [resources] sub-client
// translates that into an [ErrNotFound]-bearing [*APIError].
//
// bodyHint is preserved on the synthesized error's Body field for
// diagnostic purposes; it is not parsed.
func (a coreAdapter) SynthesizeError(op, method, requestURL string, statusCode int, bodyHint string) error {
	return newAPIError(op, method, requestURL, statusCode, []byte(bodyHint))
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
