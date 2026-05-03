// Package services is the v2 sub-client for GeoServer's per-service
// OWS configuration endpoints — `/rest/services/{wms,wfs,wcs,wmts}/settings`.
//
// The companion to [v2/rest/settings] (which covers the global
// `/rest/settings` document). This package handles the per-service
// tunables that don't live there: WFS `maxFeatures`, WMS
// `maxRenderingTime` + watermark + interpolation, WCS memory caps,
// plus per-workspace overrides for tenanted deployments.
//
// Service slugs supported here: `wms`, `wfs`, `wcs`, `wmts`. The
// `oseo` (OpenSearch for Earth Observation) extension shares the
// same wire shape and can be added in a follow-up if a real caller
// needs it.
//
// Every per-service Settings type embeds [ServiceInfo] (the common
// metadata block) and adds service-specific fields. The wire
// envelope key is the lowercase service slug — `{"wms":{...}}`,
// `{"wfs":{...}}`, etc.
package services

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ServiceInfo is the common metadata block every OWS service shares.
// Embedded into [WMSSettings], [WFSSettings], [WCSSettings], and
// [WMTSSettings].
//
// Wire-shape note: the schema spells "abstract" as `abstrct` (typo
// preserved on the wire); the Go field name is `Abstract` and the
// JSON tag captures the actual wire key.
type ServiceInfo struct {
	Enabled           bool          `json:"enabled,omitempty"`
	Name              string        `json:"name,omitempty"`
	Title             string        `json:"title,omitempty"`
	Maintainer        string        `json:"maintainer,omitempty"`
	Abstract          string        `json:"abstrct,omitempty"`
	AccessConstraints string        `json:"accessConstraints,omitempty"`
	Fees              string        `json:"fees,omitempty"`
	Versions          *Versions     `json:"versions,omitempty"`
	Keywords          *Keywords     `json:"keywords,omitempty"`
	MetadataLink      *MetadataLink `json:"metadataLink,omitempty"`
	CiteCompliant     bool          `json:"citeCompliant,omitempty"`
	OnlineResource    string        `json:"onlineResource,omitempty"`
	SchemaBaseURL     string        `json:"schemaBaseURL,omitempty"`
	Verbose           bool          `json:"verbose,omitempty"`
}

// Versions wraps the supported-version list as a flat string slice.
//
// Wire-shape note: GeoServer wraps the version list in a class-name
// key (`org.geotools.util.Version`) and collapses single-element
// arrays to a scalar object. Both shapes:
//
//	"versions": {"org.geotools.util.Version": [{"version":"1.1.1"}, {"version":"1.3.0"}]}
//	"versions": {"org.geotools.util.Version": {"version":"1.0.0"}}
//
// are decoded into Versions{List: ["1.1.1","1.3.0"]} and
// Versions{List: ["1.0.0"]} respectively. Marshal always emits the
// canonical array form so write payloads are GeoServer-compatible.
type Versions struct {
	List []string
}

const versionsKey = "org.geotools.util.Version"

type versionsWire struct {
	Org json.RawMessage `json:"org.geotools.util.Version,omitempty"`
}

type versionEntry struct {
	Version string `json:"version,omitempty"`
}

// UnmarshalJSON accepts the wire shapes documented above.
func (v *Versions) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	var w versionsWire
	if err := json.Unmarshal(data, &w); err != nil {
		return fmt.Errorf("versions: %w", err)
	}
	if len(w.Org) == 0 || bytes.Equal(w.Org, []byte("null")) {
		return nil
	}
	if w.Org[0] == '[' {
		var entries []versionEntry
		if err := json.Unmarshal(w.Org, &entries); err != nil {
			return fmt.Errorf("versions: decode array: %w", err)
		}
		v.List = make([]string, 0, len(entries))
		for _, e := range entries {
			v.List = append(v.List, e.Version)
		}
		return nil
	}
	var single versionEntry
	if err := json.Unmarshal(w.Org, &single); err != nil {
		return fmt.Errorf("versions: decode single: %w", err)
	}
	if single.Version != "" {
		v.List = []string{single.Version}
	}
	return nil
}

// MarshalJSON emits the canonical array form.
func (v Versions) MarshalJSON() ([]byte, error) {
	if len(v.List) == 0 {
		return []byte("null"), nil
	}
	entries := make([]versionEntry, len(v.List))
	for i, s := range v.List {
		entries[i] = versionEntry{Version: s}
	}
	return json.Marshal(map[string]any{versionsKey: entries})
}

// Keywords wraps the keyword list as a flat string slice.
//
// Wire-shape note: GeoServer collapses single-element keyword arrays
// to a scalar string. Both shapes:
//
//	"keywords": {"string": ["WMS","GEOSERVER"]}
//	"keywords": {"string": "WMTS"}
//
// are decoded into Keywords{Strings: [...]}. Marshal always emits the
// canonical array form.
type Keywords struct {
	Strings []string
}

type keywordsWire struct {
	String json.RawMessage `json:"string,omitempty"`
}

// UnmarshalJSON accepts both wire shapes documented above.
func (k *Keywords) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	var w keywordsWire
	if err := json.Unmarshal(data, &w); err != nil {
		return fmt.Errorf("keywords: %w", err)
	}
	if len(w.String) == 0 || bytes.Equal(w.String, []byte("null")) {
		return nil
	}
	if w.String[0] == '[' {
		var arr []string
		if err := json.Unmarshal(w.String, &arr); err != nil {
			return fmt.Errorf("keywords: decode array: %w", err)
		}
		k.Strings = arr
		return nil
	}
	var single string
	if err := json.Unmarshal(w.String, &single); err != nil {
		return fmt.Errorf("keywords: decode single: %w", err)
	}
	if single != "" {
		k.Strings = []string{single}
	}
	return nil
}

// MarshalJSON emits the canonical array form.
func (k Keywords) MarshalJSON() ([]byte, error) {
	if len(k.Strings) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(struct {
		String []string `json:"string"`
	}{String: k.Strings})
}

// MetadataLink is the optional metadata-pointer block. Most callers
// leave this nil.
//
// Wire-shape note: GeoServer sends `"metadataLink": ""` (empty string)
// when no link is configured, instead of omitting the field or
// sending `null`. The custom UnmarshalJSON treats the empty-string
// form as "unset" — the resulting *MetadataLink is non-nil but with
// all-zero fields. Callers can check `link.Type == "" && link.Content == ""`
// to detect the unset case if needed; equivalent behavior to
// `link == nil` on a freshly-decoded value where the wire was empty.
type MetadataLink struct {
	Type         string `json:"type,omitempty"`
	MetadataType string `json:"metadataType,omitempty"`
	Content      string `json:"content,omitempty"`
}

// UnmarshalJSON accepts both `{type,metadataType,content}` and the
// GeoServer empty-string-when-unset form.
func (m *MetadataLink) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		return nil
	}
	type alias MetadataLink
	return json.Unmarshal(data, (*alias)(m))
}

// WMSSettings is the WMS service config (`/services/wms/settings`,
// envelope key `wms`). Schema corresponds to the upstream `WMSInfo`
// type.
type WMSSettings struct {
	ServiceInfo
	Watermark                             *Watermark `json:"watermark,omitempty"`
	Interpolation                         string     `json:"interpolation,omitempty"`
	MaxBuffer                             int        `json:"maxBuffer,omitempty"`
	MaxRequestMemory                      int        `json:"maxRequestMemory,omitempty"`
	MaxRenderingTime                      int        `json:"maxRenderingTime,omitempty"`
	MaxRenderingErrors                    int        `json:"maxRenderingErrors,omitempty"`
	GetFeatureInfoMimeTypeCheckingEnabled bool       `json:"getFeatureInfoMimeTypeCheckingEnabled,omitempty"`
	GetMapMimeTypeCheckingEnabled         bool       `json:"getMapMimeTypeCheckingEnabled,omitempty"`
	DynamicStylingDisabled                bool       `json:"dynamicStylingDisabled,omitempty"`
}

// Watermark is the WMS-only overlay block.
//
// Position values: TOP_LEFT, TOP_CENTER, TOP_RIGHT, MID_LEFT,
// MID_CENTER, MID_RIGHT, BOT_LEFT, BOT_CENTER, BOT_RIGHT.
// Transparency is 0–255 (0 = opaque, 255 = fully transparent).
type Watermark struct {
	Enabled      bool   `json:"enabled,omitempty"`
	Position     string `json:"position,omitempty"`
	Transparency int    `json:"transparency,omitempty"`
}

// WFSSettings is the WFS service config (`/services/wfs/settings`,
// envelope key `wfs`).
//
// ServiceLevel values: BASIC, TRANSACTIONAL, COMPLETE.
type WFSSettings struct {
	ServiceInfo
	MaxFeatures             int     `json:"maxFeatures,omitempty"`
	ServiceLevel            string  `json:"serviceLevel,omitempty"`
	FeatureBounding         bool    `json:"featureBounding,omitempty"`
	EncodeFeatureMember     bool    `json:"encodeFeatureMember,omitempty"`
	CanonicalSchemaLocation bool    `json:"canonicalSchemaLocation,omitempty"`
	HitsIgnoreMaxFeatures   bool    `json:"hitsIgnoreMaxFeatures,omitempty"`
	GML                     *GMLMap `json:"gml,omitempty"`
}

// GMLMap is the per-WFS-version GML output configuration map.
type GMLMap struct {
	Entry []GMLEntry `json:"entry,omitempty"`
}

// GMLEntry is one entry in [GMLMap]: WFS version → SrsNameStyle.
type GMLEntry struct {
	Version      string `json:"@key,omitempty"`
	SrsNameStyle string `json:"$,omitempty"`
}

// WCSSettings is the WCS service config (`/services/wcs/settings`,
// envelope key `wcs`).
//
// MaxInputMemory and MaxOutputMemory: the upstream OpenAPI YAML
// types these as `boolean` — that's a documented schema bug; the
// runtime values are integers (kilobytes). Go uses int64 here.
type WCSSettings struct {
	ServiceInfo
	GMLPrefixing    bool  `json:"gmlPrefixing,omitempty"`
	LatLon          bool  `json:"latLon,omitempty"`
	MaxInputMemory  int64 `json:"maxInputMemory,omitempty"`
	MaxOutputMemory int64 `json:"maxOutputMemory,omitempty"`
}

// WMTSSettings is the WMTS service config (`/services/wmts/settings`,
// envelope key `wmts`). The upstream schema documents no unique fields
// for WMTS beyond [ServiceInfo]; this type embeds it plain.
type WMTSSettings struct {
	ServiceInfo
}

// Per-service envelope wrappers. The wire shape is
// `{"<slug>": {<settings fields>}}`.

type wmsEnvelope struct {
	WMS *WMSSettings `json:"wms"`
}

type wfsEnvelope struct {
	WFS *WFSSettings `json:"wfs"`
}

type wcsEnvelope struct {
	WCS *WCSSettings `json:"wcs"`
}

type wmtsEnvelope struct {
	WMTS *WMTSSettings `json:"wmts"`
}
