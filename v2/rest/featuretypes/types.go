// Package featuretypes is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/datastores/{ds}/featuretypes resource. The
// client is a 2-level hierarchy: callers obtain a [*DatastoreClient]
// via [Client.InWorkspace] then [WorkspaceClient.InDatastore], and all
// CRUD lives on the resulting client.
package featuretypes

import (
	"encoding/json"
	"fmt"
)

// FeatureType is the GeoServer feature-type document. The same shape is
// used for read and write paths; fields are pointer or omitempty so that
// minimal Create payloads round-trip correctly.
//
// Namespace and Store are response-only references on read paths — the
// SDK builds URLs itself rather than following the response Href, and
// a Create payload should leave both nil (the workspace and datastore
// are derived from the URL scope).
type FeatureType struct {
	Name                   string             `json:"name,omitempty"`
	NativeName             string             `json:"nativeName,omitempty"`
	Namespace              *Ref               `json:"namespace,omitempty"`
	Title                  string             `json:"title,omitempty"`
	Abstract               string             `json:"abstract,omitempty"`
	Keywords               *Keywords          `json:"keywords,omitempty"`
	MetadataLinks          *MetadataLinks     `json:"metadatalinks,omitempty"`
	DataLinks              *DataLinks         `json:"dataLinks,omitempty"`
	NativeCRS              *CRS               `json:"nativeCRS,omitempty"`
	SRS                    string             `json:"srs,omitempty"`
	Enabled                bool               `json:"enabled,omitempty"`
	NativeBoundingBox      *NativeBoundingBox `json:"nativeBoundingBox,omitempty"`
	LatLonBoundingBox      *LatLonBoundingBox `json:"latLonBoundingBox,omitempty"`
	ProjectionPolicy       string             `json:"projectionPolicy,omitempty"`
	Metadata               *Metadata          `json:"metadata,omitempty"`
	Store                  *Ref               `json:"store,omitempty"`
	CqlFilter              string             `json:"cqlFilter,omitempty"`
	MaxFeatures            int32              `json:"maxFeatures,omitempty"`
	NumDecimals            float32            `json:"numDecimals,omitempty"`
	ResponseSRS            *ResponseSRS       `json:"responseSRS,omitempty"`
	CircularArcPresent     bool               `json:"circularArcPresent,omitempty"`
	OverridingServiceSRS   bool               `json:"overridingServiceSRS,omitempty"`
	SkipNumberMatched      bool               `json:"skipNumberMatched,omitempty"`
	LinearizationTolerance bool               `json:"linearizationTolerance,omitempty"`
	Attributes             *Attributes        `json:"attributes,omitempty"`
}

// Ref is a generic reference object (name + href) carried in feature-type
// responses for namespace, store, and similar pointers. Only Name is
// meaningful for SDK callers — the SDK builds its own URLs.
type Ref struct {
	Name string `json:"name,omitempty"`
	Href string `json:"href,omitempty"`
}

// CRS models the GeoServer coordinate-reference-system field, which on
// the wire is either a JSON object ({"@class":"projected","$":"EPSG:4326"})
// or a bare string identifier. The custom Marshal / Unmarshal preserves
// this asymmetry so reads round-trip correctly.
//
// When constructing for write, set Class="" and Value="EPSG:xxxx" for
// the bare-string form; set Class="projected" / "geographic" / "string"
// (and Value to the SRS) for the object form.
type CRS struct {
	Class string `json:"@class,omitempty"`
	Value string `json:"$,omitempty"`
}

// UnmarshalJSON accepts both shapes returned by GeoServer:
//
//   - object: {"@class":"projected","$":"EPSG:4326"}
//   - bare string: "EPSG:4326"
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
			return fmt.Errorf("featuretypes: unrecognized CRS payload: %v", v)
		}
		*c = CRS{Class: class, Value: value}
	case string:
		*c = CRS{Class: "string", Value: v}
	default:
		return fmt.Errorf("featuretypes: unrecognized CRS payload type: %T", v)
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

// NativeBoundingBox is the feature-type extent in the native CRS.
type NativeBoundingBox struct {
	BoundingBox
	CRS *CRS `json:"crs,omitempty"`
}

// LatLonBoundingBox is the feature-type extent in WGS84 lat/lon.
type LatLonBoundingBox struct {
	BoundingBox
	CRS *CRS `json:"crs,omitempty"`
}

// Keywords is the keywords block on a feature-type document.
type Keywords struct {
	String []string `json:"string,omitempty"`
}

// Metadata is the loose key/value bag GeoServer uses for assorted
// feature-type metadata (e.g., "time", "elevation", "JDBC_VIRTUAL_TABLE").
type Metadata struct {
	Entry []MetadataEntry `json:"entry,omitempty"`
}

// MetadataEntry is one key/value pair inside [Metadata]. The wire form
// is `{"@key":"...","$":"..."}` (XML-as-JSON convention).
type MetadataEntry struct {
	Key   string `json:"@key"`
	Value string `json:"$"`
}

// MetadataLinks groups external metadata URLs.
type MetadataLinks struct {
	MetadataLink []MetadataLink `json:"metadataLink,omitempty"`
}

// MetadataLink is one external metadata URL.
type MetadataLink struct {
	Type         string `json:"type,omitempty"`
	MetadataType string `json:"metadataType,omitempty"`
	Content      string `json:"content,omitempty"`
}

// DataLinks groups data-distribution URLs. The JSON tag is the awkward
// implementation-class name GeoServer emits on the wire.
type DataLinks struct {
	DataLink []MetadataLink `json:"org.geoserver.catalog.impl.DataLinkInfoImpl,omitempty"`
}

// ResponseSRS is the list of EPSG codes GeoServer offers for this
// feature type in WFS responses.
type ResponseSRS struct {
	String []int `json:"string,omitempty"`
}

// Attributes wraps the attribute list.
type Attributes struct {
	Attribute []Attribute `json:"attribute,omitempty"`
}

// Attribute describes one column of the underlying source.
type Attribute struct {
	Name      string `json:"name,omitempty"`
	MinOccurs int16  `json:"minOccurs,omitempty"`
	MaxOccurs int16  `json:"maxOccurs,omitempty"`
	Nillable  bool   `json:"nillable,omitempty"`
	Binding   string `json:"binding,omitempty"`
	Length    int16  `json:"length,omitempty"`
}

// ListOptions controls listing behavior. Currently empty; the underlying
// endpoint does not paginate. Reserved for future fields.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Recurse deletes the feature type and any layers exposing it.
	// Default false rejects deletion when a layer references the
	// feature type.
	Recurse bool
}

// DiscoverOptions controls [DatastoreClient.Discover] behavior — the
// listing of feature-type names that exist in the underlying datastore
// but have not yet been published as GeoServer feature types.
type DiscoverOptions struct {
	// Kind selects which subset to return. The zero value is
	// [DiscoverAvailable].
	Kind DiscoverKind
}

// DiscoverKind selects the GeoServer `?list=…` query value.
type DiscoverKind string

// Discover modes. See [DatastoreClient.Discover] for usage.
const (
	// DiscoverAvailable lists tables in the datastore not yet
	// configured as feature types.
	DiscoverAvailable DiscoverKind = "available"
	// DiscoverAvailableWithGeometry lists available tables that
	// contain a geometry column.
	DiscoverAvailableWithGeometry DiscoverKind = "available_with_geom"
	// DiscoverAll lists configured plus available — every table
	// known to the datastore.
	DiscoverAll DiscoverKind = "all"
)

// listResponse mirrors GeoServer's `{"featureTypes":{"featureType":[…]}}`.
type listResponse struct {
	FeatureTypes struct {
		FeatureType []FeatureType `json:"featureType"`
	} `json:"featureTypes"`
}

// detailResponse mirrors GeoServer's `{"featureType":{…}}`.
type detailResponse struct {
	FeatureType FeatureType `json:"featureType"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	FeatureType FeatureType `json:"featureType"`
}

// discoverResponse mirrors GeoServer's `{"list":{"string":[…]}}` shape
// for the Discover (?list=available) endpoint.
type discoverResponse struct {
	List struct {
		String []string `json:"string"`
	} `json:"list"`
}
