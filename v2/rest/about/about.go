// Package about is the v2 sub-client for the GeoServer
// /rest/about/version resource. It surfaces a health-check (Ping) and
// the GeoServer + dependency version document.
package about

import (
	"context"
	"encoding/json"
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

// Client is the v2 about sub-client.
//
//	if err := c.About.Ping(ctx); err == nil { /* GeoServer is up */ }
//	v, _ := c.About.Version(ctx); fmt.Println(v.Resource[0].Version)
type Client struct {
	core Core
}

// New constructs the sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// VersionInfo wraps the resource list returned by /rest/about/version.
// Each entry is a versioned component (GeoServer itself, GeoTools,
// GeoWebCache, etc.).
type VersionInfo struct {
	Resource []Resource `json:"resource,omitempty"`
}

// Resource is one component in [VersionInfo]. Wire shape uses the
// XML-as-JSON `@name` attribute for the component name.
//
// Version may come back as either a JSON string ("2.28.0") or a JSON
// number (34) depending on the component — GeoTools, for example,
// reports a bare integer in some releases. The custom Unmarshal
// coerces both forms into the string field.
type Resource struct {
	Name           string `json:"@name,omitempty"`
	Version        string `json:"-"`
	BuildTimestamp string `json:"Build-Timestamp,omitempty"`
	GitRevision    string `json:"Git-Revision,omitempty"`
}

// UnmarshalJSON tolerates string-or-number Version. The other fields
// decode via the alias trick to avoid recursion.
func (r *Resource) UnmarshalJSON(data []byte) error {
	type alias Resource
	aux := struct {
		*alias
		Version json.RawMessage `json:"Version,omitempty"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Version) == 0 || string(aux.Version) == "null" {
		return nil
	}
	if aux.Version[0] == '"' {
		return json.Unmarshal(aux.Version, &r.Version)
	}
	// Number or other — preserve as raw string ("34" → "34").
	r.Version = string(aux.Version)
	return nil
}

// Ping issues a GET against /rest/about/version and returns nil if
// GeoServer responded with 2xx. Useful for liveness probes from
// orchestration layers.
//
// Returns a *APIError with the underlying status if the server
// answered with a non-2xx, or a transport error if the request never
// reached the server.
func (c *Client) Ping(ctx context.Context) error {
	const op = "About.Ping"
	u, err := c.core.URL("rest", "about", "version")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodGet, u, nil, nil, nil)
}

// Version fetches the full /rest/about/version document — a list of
// component versions (GeoServer core, GeoTools, GeoWebCache, etc.)
// with build timestamps and git revisions.
//
// Use this for richer diagnostics; for a simple "is it up" check,
// [Client.Ping] is cheaper since it discards the body.
func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	const op = "About.Version"
	u, err := c.core.URL("rest", "about", "version")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp struct {
		About VersionInfo `json:"about"`
	}
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.About, nil
}

// ManifestEntry is one row of the /rest/about/manifest output — one
// OSGi bundle / JAR shipped with the GeoServer install.
//
// Manifest fields are heterogeneous across bundles (some include
// `Bundle-License` / `Bundle-Version` / `Build-Jdk`, others ship
// minimal metadata). The SDK preserves Name as a typed field; all
// other entries land in [ManifestEntry.Fields] keyed by the
// MANIFEST.MF attribute name (e.g. "Bundle-Name", "Implementation-Version").
type ManifestEntry struct {
	// Name is the bundle name as advertised by GeoServer (the
	// `@name` JSON attribute).
	Name string `json:"-"`
	// Fields carries every other manifest attribute. Values are
	// preserved as-is (string, number, etc.) via [json.RawMessage]
	// for round-trip fidelity; helper [ManifestEntry.String]
	// extracts a string-typed value.
	Fields map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON pulls the `@name` attribute into Name and stows the
// rest of the entry's fields in Fields.
func (m *ManifestEntry) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if k == "@name" {
			_ = json.Unmarshal(v, &m.Name)
			continue
		}
		if m.Fields == nil {
			m.Fields = map[string]json.RawMessage{}
		}
		m.Fields[k] = v
	}
	return nil
}

// String returns the named field's value coerced to a string. If the
// underlying JSON value is a number/bool the string form is returned
// (so "Specification-Version":1 resolves to "1"). Returns "" if the
// field is absent.
func (m ManifestEntry) String(field string) string {
	v, ok := m.Fields[field]
	if !ok {
		return ""
	}
	if len(v) > 0 && v[0] == '"' {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			return s
		}
	}
	return string(v)
}

// ListManifestsOptions controls the [Client.Manifests] response
// filtering and field selection.
type ListManifestsOptions struct {
	// Manifest is a regex applied to bundle names; only matching
	// bundles are returned. Empty matches everything.
	Manifest string
	// Key is a regex applied to attribute names; only matching
	// attributes are returned per bundle. Empty returns all.
	Key string
	// Value is a regex applied to attribute values; only matching
	// attributes are returned. Empty returns all.
	Value string
}

// Manifests fetches the installed-bundle manifest list. Useful for
// inspecting plugin versions and build metadata.
//
// On a default GeoServer install this returns ~150 entries (every
// bundled JAR), with a payload commonly >100 KB. The SDK streams
// the response rather than buffering — the standard JSON Do path's
// 8 KiB body cap would truncate the body for any non-trivial
// install.
//
// Wire-quirk: `/rest/about/manifest` defaults to HTML when filter
// query params are present; the SDK appends `.json` to force the
// JSON wire shape. Empty result comes back as `{"about":""}` (bare
// string instead of object) and is normalized to a nil slice.
func (c *Client) Manifests(ctx context.Context, opts ListManifestsOptions) ([]ManifestEntry, error) {
	const op = "About.Manifests"
	u, err := c.core.URL("rest", "about", "manifest")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	u += ".json"
	var query map[string]string
	if opts.Manifest != "" || opts.Key != "" || opts.Value != "" {
		query = map[string]string{}
		if opts.Manifest != "" {
			query["manifest"] = opts.Manifest
		}
		if opts.Key != "" {
			query["key"] = opts.Key
		}
		if opts.Value != "" {
			query["value"] = opts.Value
		}
	}
	body, _, err := c.core.DoStream(ctx, op, http.MethodGet, u, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	var resp struct {
		About json.RawMessage `json:"about"`
	}
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", op, err)
	}
	if len(resp.About) == 0 || resp.About[0] == '"' {
		return nil, nil
	}
	var inner struct {
		Resource []ManifestEntry `json:"resource"`
	}
	if err := json.Unmarshal(resp.About, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", op, err)
	}
	return inner.Resource, nil
}

// SystemMetric is one entry in the /rest/about/system-status output.
// Categories include SYSTEM, CPU, MEMORY, SWAP, FILE_SYSTEM,
// NETWORK, SENSORS, GEOSERVER. On Linux containers without OSHI
// native libs many metrics report Available=false / Value="NOT
// AVAILABLE".
type SystemMetric struct {
	Available   bool   `json:"available"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	Identifier  string `json:"identifier,omitempty"`
	Category    string `json:"category,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	// Value is preserved as a string regardless of the underlying
	// wire shape (some metrics report "42.5", others "NOT
	// AVAILABLE", others a JSON number). Use [strconv.ParseFloat]
	// or similar to coerce when Available=true and the metric is
	// known to be numeric.
	Value string `json:"-"`
}

// UnmarshalJSON tolerates string-or-number Value forms.
func (s *SystemMetric) UnmarshalJSON(data []byte) error {
	type alias SystemMetric
	aux := struct {
		*alias
		Value json.RawMessage `json:"value,omitempty"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Value) == 0 || string(aux.Value) == "null" {
		return nil
	}
	if aux.Value[0] == '"' {
		return json.Unmarshal(aux.Value, &s.Value)
	}
	s.Value = string(aux.Value)
	return nil
}

// SystemStatus fetches the live `/rest/about/system-status` metrics.
// Each entry covers a single OS / JVM / GeoServer telemetry point;
// see [SystemMetric] for the available categories.
func (c *Client) SystemStatus(ctx context.Context) ([]SystemMetric, error) {
	const op = "About.SystemStatus"
	u, err := c.core.URL("rest", "about", "system-status")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp struct {
		Metrics struct {
			Metric []SystemMetric `json:"metric"`
		} `json:"metrics"`
	}
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Metrics.Metric, nil
}
