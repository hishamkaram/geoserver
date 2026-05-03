// Package gwc is the v2 sub-client for the GeoWebCache REST endpoints
// at `/gwc/rest/` — universal for any GeoServer deployment serving
// map tiles. The current surface covers the three groups documented
// at `/latest/en/user/geowebcache/rest/`:
//
//   - Layers — per-layer cache config (gridsets, MIME types,
//     parameter filters, enabled flag). Wire format is XML-only.
//   - Seed — submit and poll seed/reseed/truncate tasks.
//     Asynchronous: POST returns immediately; status is GET-polled.
//   - DiskQuota — disk-quota policy (LFU/LRU eviction, max disk usage).
//
// Other GWC endpoints (masstruncate, blobstores, gridsets, statistics,
// global) live in the standalone GeoWebCache project docs and aren't
// covered here yet — same wire-shape pattern, can land in a follow-up.
//
// URL prefix note: `/gwc/rest/` lives outside the v1/v2 `/rest/` tree;
// the URL builder accepts arbitrary path segments, so `c.core.URL(
// "gwc", "rest", "layers")` produces the right path.
package gwc

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// ----- Layers (XML wire format) -----

// LayerConfig is the per-layer cache configuration document
// (`<GeoServerLayer>`). Sent and received as XML via
// `/gwc/rest/layers/<layer>.xml`.
type LayerConfig struct {
	XMLName          xml.Name          `xml:"GeoServerLayer"`
	ID               string            `xml:"id,omitempty"`
	Enabled          bool              `xml:"enabled"`
	InMemoryCached   bool              `xml:"inMemoryCached,omitempty"`
	Name             string            `xml:"name"`
	MimeFormats      *MimeFormats      `xml:"mimeFormats,omitempty"`
	GridSubsets      *GridSubsets      `xml:"gridSubsets,omitempty"`
	MetaWidthHeight  *MetaWidthHeight  `xml:"metaWidthHeight,omitempty"`
	ExpireCache      int               `xml:"expireCache,omitempty"`
	ExpireClients    int               `xml:"expireClients,omitempty"`
	ParameterFilters *ParameterFilters `xml:"parameterFilters,omitempty"`
	Gutter           int               `xml:"gutter,omitempty"`
}

// MimeFormats wraps the supported tile MIME-types list.
type MimeFormats struct {
	String []string `xml:"string"`
}

// GridSubsets wraps the per-CRS gridset bindings.
type GridSubsets struct {
	GridSubset []GridSubset `xml:"gridSubset"`
}

// GridSubset is one CRS-binding entry on a layer (typically
// `EPSG:4326` and `EPSG:900913`).
type GridSubset struct {
	GridSetName string  `xml:"gridSetName"`
	MinX        float64 `xml:"extent>coords>double,omitempty"`
}

// MetaWidthHeight wraps the meta-tile size (e.g. `<int>4</int><int>4</int>`
// for a 4×4 meta-tile).
type MetaWidthHeight struct {
	Int []int `xml:"int"`
}

// ParameterFilters wraps the per-layer query-param filter list.
// The most common entry is StyleParameterFilter, which lets callers
// request alternate styles via `?STYLES=...` against the cached tile
// set.
type ParameterFilters struct {
	StyleParameterFilter []StyleParameterFilter `xml:"styleParameterFilter,omitempty"`
}

// StyleParameterFilter declares the cache-key contribution of the
// `STYLES` WMS parameter.
type StyleParameterFilter struct {
	Key          string `xml:"key"`
	DefaultValue string `xml:"defaultValue"`
}

// ----- Seed -----

// SeedRequest is the body for `POST /gwc/rest/seed/<layer>.json`.
// The wire envelope key is `seedRequest`.
//
// `Type` is one of `seed` (cache new tiles), `reseed` (regenerate
// existing tiles), or `truncate` (invalidate without regenerating).
// Use the [OpSeed], [OpReseed], [OpTruncate] constants.
//
// `Format` is the tile MIME type (e.g., `image/png`, `image/jpeg`),
// distinct from the URL's `<format>` segment which selects the
// request/response document format (json / xml).
type SeedRequest struct {
	Name        string          `json:"name"`
	SRS         SRS             `json:"srs"`
	ZoomStart   int             `json:"zoomStart"`
	ZoomStop    int             `json:"zoomStop"`
	Format      string          `json:"format"`
	Type        SeedOp          `json:"type"`
	ThreadCount int             `json:"threadCount,omitempty"`
	Bounds      *Bounds         `json:"bounds,omitempty"`
	GridSetID   string          `json:"gridSetId,omitempty"`
	Parameters  *SeedParameters `json:"parameters,omitempty"`
}

// SeedOp enumerates the documented seed-task operations.
type SeedOp string

// Seed-task operation kinds.
const (
	// OpSeed caches new tiles (no-op if already cached).
	OpSeed SeedOp = "seed"
	// OpReseed regenerates tiles whether or not they already exist.
	OpReseed SeedOp = "reseed"
	// OpTruncate invalidates the cache without regenerating tiles.
	OpTruncate SeedOp = "truncate"
)

// SRS is the spatial-reference-system identifier.
type SRS struct {
	Number int `json:"number"`
}

// Bounds is the seed task's geographic envelope. The wire shape uses
// the `coords.double[]` array form GeoServer expects.
type Bounds struct {
	Coords BoundsCoords `json:"coords"`
}

// BoundsCoords wraps the four-corner array (minX, minY, maxX, maxY).
type BoundsCoords struct {
	Double []float64 `json:"double"`
}

// SeedParameters supplies the per-task parameter filter values
// (e.g., to cache only the `polygon` style for a given layer).
type SeedParameters struct {
	Entry []SeedParameterEntry `json:"entry"`
}

// SeedParameterEntry is one (key, value) pair in a parameter map.
// On the wire this is a 2-element JSON array.
type SeedParameterEntry [2]string

// seedRequestEnvelope wraps SeedRequest in the canonical wire shape.
type seedRequestEnvelope struct {
	SeedRequest *SeedRequest `json:"seedRequest"`
}

// SeedStatus is the response from `GET /gwc/rest/seed.json` (global)
// or `GET /gwc/rest/seed/<layer>.json` (per-layer). Each entry in the
// outer array describes one running task as a fixed-position
// 5-element inner array; helper accessors decode the positional
// fields by name.
//
// Wire shape: `{"long-array-array":[[tilesProcessed, totalTiles,
// remainingSeconds, taskId, taskStatus], ...]}`. Status codes:
// -1=ABORTED, 0=PENDING, 1=RUNNING, 2=DONE.
type SeedStatus struct {
	Tasks []SeedTask
}

// SeedTask is one running or recently-finished seed task.
type SeedTask struct {
	TilesProcessed   int64
	TotalTiles       int64
	RemainingSeconds int64
	TaskID           int64
	Status           SeedTaskStatus
}

// SeedTaskStatus is the status code for a [SeedTask].
type SeedTaskStatus int

// Seed-task status codes.
const (
	StatusAborted SeedTaskStatus = -1
	StatusPending SeedTaskStatus = 0
	StatusRunning SeedTaskStatus = 1
	StatusDone    SeedTaskStatus = 2
)

// String returns a human-readable status label.
func (s SeedTaskStatus) String() string {
	switch s {
	case StatusAborted:
		return "ABORTED"
	case StatusPending:
		return "PENDING"
	case StatusRunning:
		return "RUNNING"
	case StatusDone:
		return "DONE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}

// seedStatusWire is the on-wire envelope.
type seedStatusWire struct {
	LongArrayArray [][]int64 `json:"long-array-array"`
}

// UnmarshalJSON decodes the {"long-array-array":[[...]]} wire shape
// into a flat [SeedTask] slice.
func (s *SeedStatus) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	var w seedStatusWire
	if err := json.Unmarshal(data, &w); err != nil {
		return fmt.Errorf("seed status: %w", err)
	}
	s.Tasks = make([]SeedTask, 0, len(w.LongArrayArray))
	for _, row := range w.LongArrayArray {
		var t SeedTask
		if len(row) > 0 {
			t.TilesProcessed = row[0]
		}
		if len(row) > 1 {
			t.TotalTiles = row[1]
		}
		if len(row) > 2 {
			t.RemainingSeconds = row[2]
		}
		if len(row) > 3 {
			t.TaskID = row[3]
		}
		if len(row) > 4 {
			t.Status = SeedTaskStatus(row[4])
		}
		s.Tasks = append(s.Tasks, t)
	}
	return nil
}

// ----- DiskQuota -----

// DiskQuota is the disk-quota policy at `/gwc/rest/diskquota.json`.
// Wire envelope: `{"org.geowebcache.diskquota.DiskQuotaConfig":{...}}`
// — a class-name wrapper similar to the [services.Versions] pattern.
type DiskQuota struct {
	Enabled                    bool   `json:"enabled"`
	CacheCleanUpFrequency      int    `json:"cacheCleanUpFrequency,omitempty"`
	CacheCleanUpUnits          string `json:"cacheCleanUpUnits,omitempty"`
	MaxConcurrentCleanUps      int    `json:"maxConcurrentCleanUps,omitempty"`
	GlobalExpirationPolicyName string `json:"globalExpirationPolicyName,omitempty"`
	GlobalQuota                *Quota `json:"globalQuota,omitempty"`
}

// Quota is the disk-usage cap for a [DiskQuota] config.
type Quota struct {
	ID    int   `json:"id,omitempty"`
	Bytes int64 `json:"bytes"`
}

// diskQuotaEnvelope wraps DiskQuota in the class-name wire shape on
// the GET path. Note the JSON form is read-only; the PUT path
// requires XML — see [diskQuotaPutXML].
type diskQuotaEnvelope struct {
	Config *DiskQuota `json:"org.geowebcache.diskquota.DiskQuotaConfig"`
}

// diskQuotaPutXML is the XML wire shape required by `PUT
// /gwc/rest/diskquota.xml`. The GWC server-side parser
// (`QuotaXSTreamConverter`) on PUT expects `<globalQuota>` to use
// `<value>NUMBER</value><units>UNIT</units>` rather than the
// `<bytes>NUMBER</bytes>` form GET returns. This is a known GWC
// asymmetry between the read and write parsers.
type diskQuotaPutXML struct {
	XMLName                    xml.Name        `xml:"org.geowebcache.diskquota.DiskQuotaConfig"`
	Enabled                    bool            `xml:"enabled"`
	CacheCleanUpFrequency      int             `xml:"cacheCleanUpFrequency,omitempty"`
	CacheCleanUpUnits          string          `xml:"cacheCleanUpUnits,omitempty"`
	MaxConcurrentCleanUps      int             `xml:"maxConcurrentCleanUps,omitempty"`
	GlobalExpirationPolicyName string          `xml:"globalExpirationPolicyName,omitempty"`
	GlobalQuota                *quotaPutXMLVal `xml:"globalQuota,omitempty"`
}

// quotaPutXMLVal is the value/units form GWC PUT requires.
type quotaPutXMLVal struct {
	Value int64  `xml:"value"`
	Units string `xml:"units"`
}
