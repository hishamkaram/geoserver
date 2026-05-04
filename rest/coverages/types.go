// Package coverages is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/coveragestores/{cs}/coverages resource.
// Coverages are the raster-side analogue of feature types: a coverage
// is one published raster layer derived from a coverage store.
//
// The client is a 2-level hierarchy: callers obtain a
// [*CoverageStoreClient] via [Client.InWorkspace] then
// [WorkspaceClient.InCoverageStore].
package coverages

import "github.com/hishamkaram/geoserver/v2/internal/wire"

// Shared GIS wire-format types. These are aliases for the canonical
// definitions in v2/internal/wire — the same underlying types used by
// the featuretypes sub-package, so values can flow between packages
// without conversion.
type (
	// CRS — see [wire.CRS] for marshal/unmarshal semantics.
	CRS = wire.CRS
	// BoundingBox — see [wire.BoundingBox].
	BoundingBox = wire.BoundingBox
	// NativeBoundingBox — see [wire.NativeBoundingBox].
	NativeBoundingBox = wire.NativeBoundingBox
	// LatLonBoundingBox — see [wire.LatLonBoundingBox].
	LatLonBoundingBox = wire.LatLonBoundingBox
	// Keywords — see [wire.Keywords].
	Keywords = wire.Keywords
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
