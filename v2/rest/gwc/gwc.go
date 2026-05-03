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
