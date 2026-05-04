package gwc

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
// Defined here as an interface so this subpackage doesn't import the
// root package.
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoXML(ctx context.Context, op, method, requestURL string, query map[string]string, out any) error
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
}

// Client is the v2 GeoWebCache sub-client. Reach the per-resource
// clients via the accessors:
//
//	c.GWC.Layers().List(ctx)
//	c.GWC.Seed().Submit(ctx, "topp:states", &gwc.SeedRequest{...})
//	c.GWC.DiskQuota().Get(ctx)
//
// Construct via the parent [*geoserver.Client]; do not call [New]
// directly outside the root package's wiring.
type Client struct {
	core Core
}

// New constructs the entry-point client.
func New(core Core) *Client {
	return &Client{core: core}
}

// Layers returns the per-layer cache configuration client.
func (c *Client) Layers() *LayersClient { return &LayersClient{core: c.core} }

// Seed returns the seed/reseed/truncate task client.
func (c *Client) Seed() *SeedClient { return &SeedClient{core: c.core} }

// DiskQuota returns the disk-quota policy client.
func (c *Client) DiskQuota() *DiskQuotaClient { return &DiskQuotaClient{core: c.core} }

// Global returns the singleton GWC config client.
func (c *Client) Global() *GlobalClient { return &GlobalClient{core: c.core} }

// Gridsets returns the named tile-matrix-set client.
func (c *Client) Gridsets() *GridsetsClient { return &GridsetsClient{core: c.core} }

// MassTruncate returns the bulk cache-invalidation client.
func (c *Client) MassTruncate() *MassTruncateClient { return &MassTruncateClient{core: c.core} }

// ----- Layers -----

// LayersClient covers `/gwc/rest/layers` and `/gwc/rest/layers/<layer>.xml`.
// The per-layer endpoint is XML-only; List uses the JSON form for a
// flat array of layer names.
type LayersClient struct {
	core Core
}

// List returns the names of every layer GeoWebCache knows about
// (qualified `<workspace>:<layer>` form for workspace-scoped layers).
func (c *LayersClient) List(ctx context.Context) ([]string, error) {
	const op = "GWC.Layers.List"
	u, err := c.core.URL("gwc", "rest", "layers.json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var out []string
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get fetches the per-layer cache configuration for `name` (use the
// qualified `<workspace>:<layer>` form for workspace-scoped layers,
// e.g. `topp:states`).
//
// XML-only response; returns a *APIError wrapping ErrNotFound for
// unknown layers.
func (c *LayersClient) Get(ctx context.Context, name string) (*LayerConfig, error) {
	const op = "GWC.Layers.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("gwc", "rest", "layers", name+".xml")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var out LayerConfig
	if err := c.core.DoXML(ctx, op, http.MethodGet, u, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Put replaces (or creates) the per-layer cache configuration. The
// request body is the XML-serialized [LayerConfig].
func (c *LayersClient) Put(ctx context.Context, name string, layer *LayerConfig) error {
	const op = "GWC.Layers.Put"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if layer == nil {
		return errors.New(op + ": nil layer config")
	}
	u, err := c.core.URL("gwc", "rest", "layers", name+".xml")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body, err := xml.Marshal(layer)
	if err != nil {
		return fmt.Errorf("%s: encode body: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u, bytes.NewReader(body),
		"application/xml", "*/*", nil)
}

// Delete removes the per-layer cache configuration. The catalog layer
// itself is unaffected.
func (c *LayersClient) Delete(ctx context.Context, name string) error {
	const op = "GWC.Layers.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("gwc", "rest", "layers", name+".xml")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// ----- Seed -----

// SeedClient covers `/gwc/rest/seed/...` — submit and poll
// asynchronous seed/reseed/truncate tasks.
type SeedClient struct {
	core Core
}

// Submit kicks off a new seed/reseed/truncate task on `layer`. The
// call is asynchronous: the server returns 200 immediately and runs
// the task in the background. Poll [SeedClient.Status] (per-layer)
// or [SeedClient.StatusAll] (global) for progress; cancel via
// [SeedClient.KillAll].
//
// The `name` field on req must match `layer` (GeoServer rejects
// mismatches). Pass [OpSeed], [OpReseed], or [OpTruncate] for `Type`.
func (c *SeedClient) Submit(ctx context.Context, layer string, req *SeedRequest) error {
	const op = "GWC.Seed.Submit"
	if layer == "" {
		return errors.New(op + ": empty layer name")
	}
	if req == nil {
		return errors.New(op + ": nil seed request")
	}
	if req.Name == "" {
		req.Name = layer
	}
	if req.Type == "" {
		return errors.New(op + ": empty SeedRequest.Type (want seed | reseed | truncate)")
	}
	u, err := c.core.URL("gwc", "rest", "seed", layer+".json")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := seedRequestEnvelope{SeedRequest: req}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Status returns the running-tasks list for the named layer.
func (c *SeedClient) Status(ctx context.Context, layer string) (*SeedStatus, error) {
	const op = "GWC.Seed.Status"
	if layer == "" {
		return nil, errors.New(op + ": empty layer name")
	}
	u, err := c.core.URL("gwc", "rest", "seed", layer+".json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var out SeedStatus
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StatusAll returns the running-tasks list across every layer. Each
// task's TaskID is unique server-wide.
func (c *SeedClient) StatusAll(ctx context.Context) (*SeedStatus, error) {
	const op = "GWC.Seed.StatusAll"
	u, err := c.core.URL("gwc", "rest", "seed.json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var out SeedStatus
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// KillAll terminates every running seed task. Wire form is `POST
// /gwc/rest/seed` with `kill_all=all` as a form parameter.
func (c *SeedClient) KillAll(ctx context.Context) error {
	const op = "GWC.Seed.KillAll"
	u, err := c.core.URL("gwc", "rest", "seed")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPost, u,
		bytes.NewReader([]byte("kill_all=all")),
		"application/x-www-form-urlencoded", "*/*", nil)
}

// ----- DiskQuota -----

// DiskQuotaClient covers `/gwc/rest/diskquota.json`.
type DiskQuotaClient struct {
	core Core
}

// Get returns the current disk-quota policy.
func (c *DiskQuotaClient) Get(ctx context.Context) (*DiskQuota, error) {
	const op = "GWC.DiskQuota.Get"
	u, err := c.core.URL("gwc", "rest", "diskquota.json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var env diskQuotaEnvelope
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &env); err != nil {
		return nil, err
	}
	if env.Config == nil {
		env.Config = &DiskQuota{}
	}
	return env.Config, nil
}

// Update writes the disk-quota policy. Disk-quota is global (not
// per-layer); the change applies to all cached tiles immediately.
//
// Wire-format quirk handled here: GWC's read endpoint accepts JSON
// with `globalQuota.bytes`, but its PUT endpoint goes through a
// different parser (`QuotaXSTreamConverter`) that requires XML and
// uses `<globalQuota><value>N</value><units>B</units></globalQuota>`.
// The Update method translates [Quota.Bytes] into the XML
// value/units form (always serializing as `B` bytes for fidelity)
// and sends via XML.
func (c *DiskQuotaClient) Update(ctx context.Context, dq *DiskQuota) error {
	const op = "GWC.DiskQuota.Update"
	if dq == nil {
		return errors.New(op + ": nil disk quota")
	}
	u, err := c.core.URL("gwc", "rest", "diskquota.xml")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	xmlForm := diskQuotaPutXML{
		Enabled:                    dq.Enabled,
		CacheCleanUpFrequency:      dq.CacheCleanUpFrequency,
		CacheCleanUpUnits:          dq.CacheCleanUpUnits,
		MaxConcurrentCleanUps:      dq.MaxConcurrentCleanUps,
		GlobalExpirationPolicyName: dq.GlobalExpirationPolicyName,
	}
	if dq.GlobalQuota != nil {
		xmlForm.GlobalQuota = &quotaPutXMLVal{
			Value: dq.GlobalQuota.Bytes,
			Units: "B",
		}
	}
	body, err := xml.Marshal(xmlForm)
	if err != nil {
		return fmt.Errorf("%s: encode body: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u, bytes.NewReader(body),
		"application/xml", "*/*", nil)
}

// ----- Global -----

// GlobalClient covers the singleton `/gwc/rest/global` endpoint —
// runtime stats toggle, WMTS CITE compliance flag, backend timeout,
// and read-only identifier / location / version metadata.
type GlobalClient struct {
	core Core
}

// Get returns the current global GWC configuration.
func (c *GlobalClient) Get(ctx context.Context) (*Global, error) {
	const op = "GWC.Global.Get"
	u, err := c.core.URL("gwc", "rest", "global.json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var g Global
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// Update writes the global GWC configuration. Identifier, Location,
// and Version are server-managed read-only fields — callers may leave
// them at their Get-supplied values; the server preserves them.
func (c *GlobalClient) Update(ctx context.Context, g *Global) error {
	const op = "GWC.Global.Update"
	if g == nil {
		return errors.New(op + ": nil global config")
	}
	u, err := c.core.URL("gwc", "rest", "global.json")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, g, nil, nil)
}

// ----- Gridsets -----

// GridsetsClient covers `/gwc/rest/gridsets` — list, fetch, and
// delete named tile-matrix sets. Create is deferred (XML wire shape
// for arbitrary CRS extents is gnarly; the built-in gridsets cover
// EPSG:4326, WebMercatorQuad, and dozens of UTM tilings out of the
// box).
type GridsetsClient struct {
	core Core
}

// List returns the names of every defined gridset.
func (c *GridsetsClient) List(ctx context.Context) ([]string, error) {
	const op = "GWC.Gridsets.List"
	u, err := c.core.URL("gwc", "rest", "gridsets.json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var out []string
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get fetches a single gridset definition. Returns a *APIError
// wrapping ErrNotFound for unknown names.
func (c *GridsetsClient) Get(ctx context.Context, name string) (*GridSet, error) {
	const op = "GWC.Gridsets.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("gwc", "rest", "gridsets", name+".json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var env gridSetEnvelope
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &env); err != nil {
		return nil, err
	}
	if env.GridSet == nil {
		env.GridSet = &GridSet{}
	}
	return env.GridSet, nil
}

// Delete removes a custom gridset. The built-in gridsets (EPSG:4326,
// WebMercatorQuad, etc.) are protected by the server and return an
// error on delete.
func (c *GridsetsClient) Delete(ctx context.Context, name string) error {
	const op = "GWC.Gridsets.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("gwc", "rest", "gridsets", name+".xml")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// ----- MassTruncate -----

// MassTruncateClient covers `/gwc/rest/masstruncate` — invalidate
// caches in bulk by layer, parameter permutation, orphan, or extent.
//
// Wire-quirk: the endpoint requires Content-Type "text/xml"; the
// XStream parser registered under "application/xml" rejects the
// request body with "Format extension unknown".
type MassTruncateClient struct {
	core Core
}

// Capabilities returns the four documented mass-truncate operation
// kinds. Useful for cross-version probing — newer GeoServer versions
// may add operations.
func (c *MassTruncateClient) Capabilities(ctx context.Context) ([]MassTruncateRequestType, error) {
	const op = "GWC.MassTruncate.Capabilities"
	u, err := c.core.URL("gwc", "rest", "masstruncate.xml")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	type wire struct {
		XMLName     xml.Name `xml:"massTruncateRequests"`
		RequestType []string `xml:"requestType"`
	}
	var w wire
	if err := c.core.DoXML(ctx, op, http.MethodGet, u, nil, &w); err != nil {
		return nil, err
	}
	out := make([]MassTruncateRequestType, len(w.RequestType))
	for i, t := range w.RequestType {
		out[i] = MassTruncateRequestType(t)
	}
	return out, nil
}

// TruncateLayer clears every cache (all gridsets, parameter
// permutations, image formats) for the named layer.
func (c *MassTruncateClient) TruncateLayer(ctx context.Context, layerName string) error {
	const op = "GWC.MassTruncate.TruncateLayer"
	if layerName == "" {
		return errors.New(op + ": empty layerName")
	}
	return c.post(ctx, op, &MassTruncateLayerRequest{LayerName: layerName})
}

// TruncateParameters removes cache entries for parameter
// permutations no longer registered as parameter filters on the layer.
func (c *MassTruncateClient) TruncateParameters(ctx context.Context, layerName string) error {
	const op = "GWC.MassTruncate.TruncateParameters"
	if layerName == "" {
		return errors.New(op + ": empty layerName")
	}
	return c.post(ctx, op, &MassTruncateParametersRequest{LayerName: layerName})
}

// TruncateOrphans removes cache entries for layers that no longer
// exist in the catalog. Argument-less; GeoServer scans the cache
// directory.
func (c *MassTruncateClient) TruncateOrphans(ctx context.Context) error {
	const op = "GWC.MassTruncate.TruncateOrphans"
	return c.post(ctx, op, &MassTruncateOrphansRequest{})
}

// TruncateExtent removes cache entries inside an explicit bounding
// box on a named gridset. LayerName and req.Bounds are required;
// gridSetId / format / zoom range are optional filters.
func (c *MassTruncateClient) TruncateExtent(ctx context.Context, req *MassTruncateExtentRequest) error {
	const op = "GWC.MassTruncate.TruncateExtent"
	if req == nil {
		return errors.New(op + ": nil request")
	}
	if req.LayerName == "" {
		return errors.New(op + ": empty LayerName")
	}
	if req.Bounds == nil {
		return errors.New(op + ": nil Bounds")
	}
	return c.post(ctx, op, req)
}

func (c *MassTruncateClient) post(ctx context.Context, op string, req any) error {
	u, err := c.core.URL("gwc", "rest", "masstruncate")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body, err := xml.Marshal(req)
	if err != nil {
		return fmt.Errorf("%s: encode body: %w", op, err)
	}
	// Content-Type must be text/xml — application/xml dispatches to
	// a different parser that rejects the body with
	// "Format extension unknown".
	return c.core.DoRaw(ctx, op, http.MethodPost, u, bytes.NewReader(body),
		"text/xml", "*/*", nil)
}
