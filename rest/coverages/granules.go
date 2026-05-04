package coverages

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// GranulesClient operates on the granule index of a structured
// coverage (e.g., an image mosaic). The endpoint base is
// /rest/workspaces/{ws}/coveragestores/{cs}/coverages/{cov}/index/granules.
//
// Granules are the per-raster entries inside a structured coverage
// store — one row per source GeoTIFF (or other raster) that
// participates in the mosaic. Use this client to list / inspect /
// remove granules. To ADD a granule, call
// [coveragestores.WorkspaceClient.HarvestGranule] on the parent
// store; the granule index is auto-updated.
type GranulesClient struct {
	core      Core
	workspace string
	store     string
	coverage  string
}

// Granules returns a sub-client scoped to one coverage's granule
// index. coverageName is the published coverage name under the
// current store.
func (c *CoverageStoreClient) Granules(coverageName string) *GranulesClient {
	return &GranulesClient{
		core:      c.core,
		workspace: c.workspace,
		store:     c.store,
		coverage:  coverageName,
	}
}

// Workspace returns the workspace bound to this client.
func (g *GranulesClient) Workspace() string { return g.workspace }

// CoverageStore returns the coverage store bound to this client.
func (g *GranulesClient) CoverageStore() string { return g.store }

// Coverage returns the coverage bound to this client.
func (g *GranulesClient) Coverage() string { return g.coverage }

// PurgeMode controls whether the underlying raster files on disk are
// preserved or removed when granules are deleted from a structured
// coverage store. See per-constant doc.
type PurgeMode string

// Recognized purge modes for granule deletion.
const (
	// PurgeNone leaves both data and auxiliary files in place;
	// only the granule's registration in the mosaic index is
	// removed.
	PurgeNone PurgeMode = "none"
	// PurgeMetadata removes auxiliary files and metadata (e.g.
	// NetCDF sidecar indexes) but preserves the data file.
	// Recommended when the underlying raster should not be
	// deleted from disk.
	PurgeMetadata PurgeMode = "metadata"
	// PurgeAll removes both the registration and the underlying
	// raster files.
	PurgeAll PurgeMode = "all"
)

// ListGranulesOptions controls the granule list / iter calls.
type ListGranulesOptions struct {
	// Filter is an optional CQL filter to narrow the returned
	// granules (e.g. `location LIKE '%2008%'` or `BBOX(the_geom, ...)`).
	Filter string
	// Offset is the start index for paging. 0 returns from the
	// beginning. Negative values are clamped to 0 by the server.
	Offset int
	// Limit caps the number of granules returned per request. 0
	// (the default) leaves the server to choose. Useful with the
	// per-page tuning of [GranulesClient.Iter].
	Limit int
}

// DeleteGranuleOptions controls deletion of a single granule.
type DeleteGranuleOptions struct {
	// Purge controls whether the underlying raster file is
	// preserved. Empty defaults to [PurgeNone].
	Purge PurgeMode
	// UpdateBBox triggers re-calculation of the layer's native
	// bbox after the deletion.
	UpdateBBox bool
}

// DeleteGranulesOptions controls bulk deletion via CQL filter.
type DeleteGranulesOptions struct {
	// Filter is a CQL filter selecting granules to remove. Empty
	// filter deletes ALL granules — the server treats it as
	// match-all. To prevent accidental wipe, this client requires
	// a non-empty Filter — pass Filter:"INCLUDE" to delete every
	// granule deliberately.
	Filter string
	// Purge — see [DeleteGranuleOptions.Purge].
	Purge PurgeMode
	// UpdateBBox — see [DeleteGranuleOptions.UpdateBBox].
	UpdateBBox bool
}

// Granule is one entry in a structured coverage's granule index. The
// shape mirrors a GeoJSON Feature: an opaque ID, a raw geometry, and
// a free-form property map (the granule's attribute values, e.g.
// `location`, `ingestion`, `elevation`). Geometry is kept as raw
// JSON so callers can decode into the GeoJSON library of their
// choice without the SDK forcing a particular geometry type system.
type Granule struct {
	ID         string                 `json:"id,omitempty"`
	Geometry   json.RawMessage        `json:"geometry,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// granulesWire is the GeoJSON FeatureCollection envelope GeoServer
// returns from the granules endpoint.
type granulesWire struct {
	Features []Granule `json:"features"`
}

// GranuleSchema describes the granule index attributes — analogous to
// a feature type's attribute list, applied to a structured coverage.
// Use [GranulesClient.Schema] to fetch it.
type GranuleSchema struct {
	Attributes []GranuleAttribute
	// Href is the absolute URL GeoServer reports for the granules
	// list endpoint (informational; the SDK builds its own URLs).
	Href string
}

// GranuleAttribute is one attribute entry in a [GranuleSchema].
type GranuleAttribute struct {
	Name      string
	MinOccurs int
	MaxOccurs int
	Nillable  bool
	Binding   string
	Length    int
}

// schemaWire is GeoServer's JSON envelope for the granule schema:
// `{"Schema":{"attributes":{"Attribute":[{...}, ...]}, "href":"..."}}`
type schemaWire struct {
	Schema struct {
		Attributes struct {
			Attribute []GranuleAttribute `json:"Attribute"`
		} `json:"attributes"`
		Href string `json:"href"`
	} `json:"Schema"`
}

// granulesPath builds the URL segments for the granules base
// (without a trailing granule ID).
func (g *GranulesClient) granulesPath() []string {
	return []string{
		"rest", "workspaces", g.workspace,
		"coveragestores", g.store,
		"coverages", g.coverage,
		"index", "granules",
	}
}

// indexPath builds the URL segments for the index endpoint (granule
// schema).
func (g *GranulesClient) indexPath() []string {
	return []string{
		"rest", "workspaces", g.workspace,
		"coveragestores", g.store,
		"coverages", g.coverage,
		"index",
	}
}

// Schema returns the attribute schema for the granule index.
func (g *GranulesClient) Schema(ctx context.Context) (*GranuleSchema, error) {
	const op = "Coverages.Granules.Schema"
	u, err := g.core.URL(g.indexPath()...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw schemaWire
	if err := g.core.Do(ctx, op, http.MethodGet, u, nil, nil, &raw); err != nil {
		return nil, err
	}
	return &GranuleSchema{
		Attributes: raw.Schema.Attributes.Attribute,
		Href:       raw.Schema.Href,
	}, nil
}

// List returns granules under the configured coverage. To handle
// large mosaics naturally with paging, use [GranulesClient.Iter].
func (g *GranulesClient) List(ctx context.Context, opts ListGranulesOptions) ([]Granule, error) {
	const op = "Coverages.Granules.List"
	u, err := g.core.URL(g.granulesPath()...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw granulesWire
	if err := g.core.Do(ctx, op, http.MethodGet, u, nil, listQuery(opts), &raw); err != nil {
		return nil, err
	}
	return raw.Features, nil
}

// Get fetches a single granule by ID. The returned [*Granule] is
// nil if the wire FeatureCollection comes back empty (which the
// server uses for "not found" on some 2.x versions instead of 404).
// errors.Is(err, geoserver.ErrNotFound) covers the canonical 404 case.
func (g *GranulesClient) Get(ctx context.Context, granuleID string) (*Granule, error) {
	const op = "Coverages.Granules.Get"
	parts := append(g.granulesPath(), granuleID)
	u, err := g.core.URL(parts...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw granulesWire
	if err := g.core.Do(ctx, op, http.MethodGet, u, nil, nil, &raw); err != nil {
		return nil, err
	}
	if len(raw.Features) == 0 {
		return nil, nil
	}
	gr := raw.Features[0]
	return &gr, nil
}

// Delete removes a single granule by ID.
func (g *GranulesClient) Delete(ctx context.Context, granuleID string, opts DeleteGranuleOptions) error {
	const op = "Coverages.Granules.Delete"
	parts := append(g.granulesPath(), granuleID)
	u, err := g.core.URL(parts...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return g.core.Do(ctx, op, http.MethodDelete, u, nil, deleteOneQuery(opts), nil)
}

// DeleteByFilter removes every granule matching the supplied CQL
// filter. The empty filter is rejected to prevent accidental
// match-all deletions; pass Filter:"INCLUDE" to delete every
// granule deliberately.
func (g *GranulesClient) DeleteByFilter(ctx context.Context, opts DeleteGranulesOptions) error {
	const op = "Coverages.Granules.DeleteByFilter"
	if opts.Filter == "" {
		return fmt.Errorf("%s: refusing to delete all granules: pass DeleteGranulesOptions{Filter:%q} for a deliberate match-all", op, "INCLUDE")
	}
	u, err := g.core.URL(g.granulesPath()...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return g.core.Do(ctx, op, http.MethodDelete, u, nil, deleteAllQuery(opts), nil)
}

// listQuery converts ListGranulesOptions to the query map.
func listQuery(opts ListGranulesOptions) map[string]string {
	q := map[string]string{}
	if opts.Filter != "" {
		q["filter"] = opts.Filter
	}
	if opts.Offset > 0 {
		q["offset"] = strconv.Itoa(opts.Offset)
	}
	if opts.Limit > 0 {
		q["limit"] = strconv.Itoa(opts.Limit)
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// deleteOneQuery converts DeleteGranuleOptions to the query map.
func deleteOneQuery(opts DeleteGranuleOptions) map[string]string {
	q := map[string]string{}
	if opts.Purge != "" {
		q["purge"] = string(opts.Purge)
	}
	if opts.UpdateBBox {
		q["updateBBox"] = "true"
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// deleteAllQuery converts DeleteGranulesOptions to the query map.
func deleteAllQuery(opts DeleteGranulesOptions) map[string]string {
	q := map[string]string{"filter": opts.Filter}
	if opts.Purge != "" {
		q["purge"] = string(opts.Purge)
	}
	if opts.UpdateBBox {
		q["updateBBox"] = "true"
	}
	return q
}
