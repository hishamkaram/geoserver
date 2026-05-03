// Package coverages is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/coveragestores/{cs}/coverages resource.
// Coverages are the raster-side analogue of feature types: a coverage
// is one published raster layer derived from a coverage store.
//
// The client is a 2-level hierarchy: callers obtain a
// [*CoverageStoreClient] via [Client.InWorkspace] then
// [WorkspaceClient.InCoverageStore].
//
// Some types (CRS, BoundingBox, etc.) are intentionally duplicated
// from rest/featuretypes/types.go since they are GeoServer-wide GIS
// concepts. A future PR may extract them into a shared package to
// remove this duplication.
package coverages

import (
	"encoding/json"
	"fmt"
)

// Coverage is the GeoServer coverage document — one published raster
// layer. The same shape is used for read and write paths.
//
// Namespace and Store are response-only references on read paths.
// A Create payload typically only sets Name and NativeCoverageName
// (the source raster GeoServer will publish from).
type Coverage struct {
	Name                 string             `json:"name,omitempty"`
	NativeCoverageName   string             `json:"nativeCoverageName,omitempty"`
	NativeName           string             `json:"nativeName,omitempty"`
	NativeFormat         string             `json:"nativeFormat,omitempty"`
	Namespace            *Ref               `json:"namespace,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Abstract             string             `json:"abstract,omitempty"`
	Keywords             *Keywords          `json:"keywords,omitempty"`
	NativeCRS            *CRS               `json:"nativeCRS,omitempty"`
	SRS                  string             `json:"srs,omitempty"`
	Enabled              bool               `json:"enabled,omitempty"`
	NativeBoundingBox    *NativeBoundingBox `json:"nativeBoundingBox,omitempty"`
	LatLonBoundingBox    *LatLonBoundingBox `json:"latLonBoundingBox,omitempty"`
	ProjectionPolicy     string             `json:"projectionPolicy,omitempty"`
	Store                *Ref               `json:"store,omitempty"`
	CqlFilter            string             `json:"cqlFilter,omitempty"`
	OverridingServiceSRS bool               `json:"overridingServiceSRS,omitempty"`
}

// Ref is a generic reference object (name + href) carried in coverage
// responses for namespace and store pointers. Only Name is meaningful
// for SDK callers.
type Ref struct {
	Name string `json:"name,omitempty"`
	Href string `json:"href,omitempty"`
}

// CRS models the GeoServer coordinate-reference-system field, which on
// the wire is either a JSON object ({"@class":"projected","$":"EPSG:4326"})
// or a bare string identifier. The custom Marshal / Unmarshal preserves
// this asymmetry so reads round-trip correctly.
type CRS struct {
	Class string `json:"@class,omitempty"`
	Value string `json:"$,omitempty"`
}

// UnmarshalJSON accepts both the object form and the bare-string form.
func (c *CRS) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case map[string]any:
		class, _ := v["@class"].(string)
		value, _ := v["$"].(string)
		if class == "" && value == "" {
			return fmt.Errorf("coverages: unrecognized CRS payload: %v", v)
		}
		*c = CRS{Class: class, Value: value}
	case string:
		*c = CRS{Class: "string", Value: v}
	default:
		return fmt.Errorf("coverages: unrecognized CRS payload type: %T", v)
	}
	return nil
}

// MarshalJSON emits either a bare string (when Class=="string") or the
// object form. An empty CRS marshals to an empty string to match the
// wire shape GeoServer accepts on write.
func (c *CRS) MarshalJSON() ([]byte, error) {
	if c == nil || (c.Class == "" && c.Value == "") {
		return json.Marshal("")
	}
	if c.Class == "string" {
		return json.Marshal(c.Value)
	}
	return json.Marshal(struct {
		Class string `json:"@class,omitempty"`
		Value string `json:"$,omitempty"`
	}{Class: c.Class, Value: c.Value})
}

// BoundingBox is the geographic extent shared by [NativeBoundingBox]
// and [LatLonBoundingBox].
type BoundingBox struct {
	MinX float64 `json:"minx"`
	MaxX float64 `json:"maxx"`
	MinY float64 `json:"miny"`
	MaxY float64 `json:"maxy"`
}

// NativeBoundingBox is the coverage extent in the native CRS.
type NativeBoundingBox struct {
	BoundingBox
	CRS *CRS `json:"crs,omitempty"`
}

// LatLonBoundingBox is the coverage extent in WGS84 lat/lon.
type LatLonBoundingBox struct {
	BoundingBox
	CRS *CRS `json:"crs,omitempty"`
}

// Keywords is the keywords block on a coverage document.
type Keywords struct {
	String []string `json:"string,omitempty"`
}

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Recurse deletes the coverage and any layers exposing it.
	Recurse bool
}

// DiscoverOptions controls [CoverageStoreClient.Discover] behavior —
// the listing of native coverage names that exist in the underlying
// coverage store but have not yet been published as configured
// coverages.
type DiscoverOptions struct {
	// Kind selects which subset to return. The zero value is
	// [DiscoverAll] to match GeoServer's typical default for the
	// raster discovery flow (a coverage store often exposes a single
	// coverage that's already configured, so "available" returns
	// nothing and "all" is the more useful default).
	Kind DiscoverKind
}

// DiscoverKind selects the GeoServer `?list=…` query value.
type DiscoverKind string

// Discover modes. See [CoverageStoreClient.Discover] for usage.
const (
	// DiscoverAvailable lists native coverages in the store not yet
	// configured.
	DiscoverAvailable DiscoverKind = "available"
	// DiscoverAll lists configured plus available — every coverage
	// known to the store.
	DiscoverAll DiscoverKind = "all"
)

// listResponse mirrors GeoServer's `{"coverages":{"coverage":[…]}}`.
type listResponse struct {
	Coverages struct {
		Coverage []Coverage `json:"coverage"`
	} `json:"coverages"`
}

// detailResponse mirrors GeoServer's `{"coverage":{…}}`.
type detailResponse struct {
	Coverage Coverage `json:"coverage"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	Coverage Coverage `json:"coverage"`
}

// discoverResponse mirrors GeoServer's `{"list":{"string":[…]}}` shape
// for the Discover (?list=available) endpoint.
type discoverResponse struct {
	List struct {
		String []string `json:"string"`
	} `json:"list"`
}
