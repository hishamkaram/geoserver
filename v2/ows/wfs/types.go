// Package wfs is the v2 sub-client for the GeoServer WFS service.
// Covers the GetCapabilities endpoint — fetching the XML capabilities
// document and parsing it into Go types.
//
// The GeoServer WFS GetCapabilities response uses both `wfs:` and
// `ows:` XML namespaces; the type definitions in this package match
// on local name only, so values flow through Go's encoding/xml
// decoder regardless of which namespace prefix the server applied.
//
// Decoded fields are an opinionated subset of the OGC WFS 1.1.0 / 2.0
// schemas — everything callers commonly need (service identification,
// supported operations, feature type list with bounding boxes and
// SRS lists). Less-used fields (full OGC Filter capabilities, value
// constraints) are skipped on purpose to keep the type tree
// readable; add when a real caller needs them.
package wfs

import "encoding/xml"

// Capabilities is the root of the WFS GetCapabilities document.
// GeoServer emits `<wfs:WFS_Capabilities>` for both 1.1.0 and 2.0;
// the XMLName matches by local name so namespace differences don't
// trip the decoder.
type Capabilities struct {
	XMLName               xml.Name              `xml:"WFS_Capabilities"`
	Version               string                `xml:"version,attr,omitempty"`
	UpdateSequence        string                `xml:"updateSequence,attr,omitempty"`
	ServiceIdentification ServiceIdentification `xml:"ServiceIdentification"`
	ServiceProvider       ServiceProvider       `xml:"ServiceProvider"`
	OperationsMetadata    OperationsMetadata    `xml:"OperationsMetadata"`
	FeatureTypeList       FeatureTypeList       `xml:"FeatureTypeList"`
}

// ServiceIdentification carries the service-level metadata block
// (title, abstract, keywords, fees, access constraints, supported
// versions).
type ServiceIdentification struct {
	Title             string   `xml:"Title"`
	Abstract          string   `xml:"Abstract"`
	Keywords          []string `xml:"Keywords>Keyword"`
	ServiceType       string   `xml:"ServiceType"`
	Versions          []string `xml:"ServiceTypeVersion"`
	Fees              string   `xml:"Fees"`
	AccessConstraints string   `xml:"AccessConstraints"`
}

// ServiceProvider carries the provider organization metadata.
type ServiceProvider struct {
	ProviderName   string         `xml:"ProviderName"`
	OnlineResource OnlineResource `xml:"ProviderSite"`
	ServiceContact ServiceContact `xml:"ServiceContact"`
}

// ServiceContact is the provider's point-of-contact block.
type ServiceContact struct {
	IndividualName string `xml:"IndividualName"`
	PositionName   string `xml:"PositionName"`
}

// OnlineResource is the xlink:href + xlink:type pair returned for
// any linkable element (provider site, operation DCP endpoints, …).
type OnlineResource struct {
	Type string `xml:"http://www.w3.org/1999/xlink type,attr,omitempty"`
	Href string `xml:"http://www.w3.org/1999/xlink href,attr,omitempty"`
}

// OperationsMetadata enumerates the WFS operations the server
// advertises along with each operation's DCP endpoints, parameters,
// and constraints.
type OperationsMetadata struct {
	Operation []Operation `xml:"Operation"`
}

// Operation describes one server operation (GetCapabilities,
// DescribeFeatureType, GetFeature, …).
type Operation struct {
	Name       string      `xml:"name,attr"`
	DCP        []DCP       `xml:"DCP"`
	Parameter  []Parameter `xml:"Parameter"`
	Constraint []Parameter `xml:"Constraint"`
}

// DCP describes one Distributed Computing Platform (transport)
// binding for an Operation.
type DCP struct {
	HTTP HTTP `xml:"HTTP"`
}

// HTTP wraps the GET / POST endpoint URLs.
type HTTP struct {
	Get  []OnlineResource `xml:"Get"`
	Post []OnlineResource `xml:"Post"`
}

// Parameter describes one named parameter or constraint with its
// allowed values. Used for both `<Parameter>` and `<Constraint>`
// children of an Operation — they share wire shape.
type Parameter struct {
	Name          string    `xml:"name,attr"`
	AllowedValues []string  `xml:"AllowedValues>Value"`
	DefaultValue  string    `xml:"DefaultValue"`
	NoValues      *struct{} `xml:"NoValues,omitempty"`
}

// FeatureTypeList wraps the list of published feature types.
type FeatureTypeList struct {
	FeatureType []FeatureType `xml:"FeatureType"`
}

// FeatureType is one published feature type entry.
type FeatureType struct {
	Name             string           `xml:"Name"`
	Title            string           `xml:"Title"`
	Abstract         string           `xml:"Abstract"`
	Keywords         []string         `xml:"Keywords>Keyword"`
	DefaultSRS       string           `xml:"DefaultSRS"`
	OtherSRS         []string         `xml:"OtherSRS"`
	OutputFormats    []string         `xml:"OutputFormats>Format"`
	WGS84BoundingBox WGS84BoundingBox `xml:"WGS84BoundingBox"`
	MetadataURL      []MetadataURL    `xml:"MetadataURL"`
}

// WGS84BoundingBox is a geographic bounding box in EPSG:4326. The
// wire form is `<LowerCorner>minX minY</LowerCorner>` and
// `<UpperCorner>maxX maxY</UpperCorner>` — pre-split corner strings
// so callers can parse with strconv.ParseFloat as needed (kept as
// strings to avoid silent precision loss when the server emits
// values like `-180.0000000000000001`).
type WGS84BoundingBox struct {
	LowerCorner string `xml:"LowerCorner"`
	UpperCorner string `xml:"UpperCorner"`
}

// MetadataURL is a layer-level pointer to an external metadata
// document.
type MetadataURL struct {
	Type   string `xml:"type,attr,omitempty"`
	Format string `xml:"format,attr,omitempty"`
	Href   string `xml:"http://www.w3.org/1999/xlink href,attr,omitempty"`
}
