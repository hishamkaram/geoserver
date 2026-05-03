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

// FeatureSchema is the root of a WFS DescribeFeatureType response —
// an XSD schema document (`<xsd:schema>`) describing the published
// feature type's attributes.
//
// The type tree intentionally models a useful subset of XSD rather
// than the full schema language; the common WFS-emitted shape is
// `complexType > complexContent > extension > sequence > element*`,
// and the [FeatureSchema.Attributes] helper walks that tree to
// surface a flat list of attributes.
type FeatureSchema struct {
	XMLName         xml.Name        `xml:"schema"`
	TargetNamespace string          `xml:"targetNamespace,attr,omitempty"`
	Imports         []SchemaImport  `xml:"import"`
	Elements        []SchemaElement `xml:"element"`
	ComplexTypes    []ComplexType   `xml:"complexType"`
}

// SchemaImport is `<xsd:import>` — a reference to another schema
// (typically GML).
type SchemaImport struct {
	Namespace      string `xml:"namespace,attr,omitempty"`
	SchemaLocation string `xml:"schemaLocation,attr,omitempty"`
}

// SchemaElement is `<xsd:element>` at the top of a [FeatureSchema] —
// names the published feature element and points at its
// [ComplexType].
type SchemaElement struct {
	Name              string `xml:"name,attr,omitempty"`
	Type              string `xml:"type,attr,omitempty"`
	SubstitutionGroup string `xml:"substitutionGroup,attr,omitempty"`
}

// ComplexType is `<xsd:complexType>` — names the type and carries
// either a top-level [Sequence] or, more commonly for WFS, a
// [ComplexContent] wrapping an extension of `gml:AbstractFeatureType`.
type ComplexType struct {
	Name           string          `xml:"name,attr,omitempty"`
	Sequence       *Sequence       `xml:"sequence,omitempty"`
	ComplexContent *ComplexContent `xml:"complexContent,omitempty"`
}

// ComplexContent is `<xsd:complexContent>`. Its [Extension] points at
// a base type (typically `gml:AbstractFeatureType`) and adds a
// [Sequence] of attribute elements.
type ComplexContent struct {
	Extension Extension `xml:"extension"`
}

// Extension is `<xsd:extension>` — names the base type and adds a
// [Sequence] of attribute elements.
type Extension struct {
	Base     string   `xml:"base,attr,omitempty"`
	Sequence Sequence `xml:"sequence"`
}

// Sequence is `<xsd:sequence>` — the ordered list of attribute
// elements making up a feature.
type Sequence struct {
	Elements []Attribute `xml:"element"`
}

// Attribute is one feature attribute — `<xsd:element>` inside the
// sequence under a [ComplexType] (directly or via [Extension]).
//
// Type is the xsd:type string (e.g., `xsd:string`, `xsd:int`,
// `gml:Point`, `gml:MultiSurfacePropertyType`). Nillable defaults
// to false when the attribute is absent. MinOccurs is "0" for
// optional attributes (the XSD default for `<xsd:element>` inside
// a sequence) — kept as string to preserve the exact wire form.
type Attribute struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr,omitempty"`
	Nillable  bool   `xml:"nillable,attr,omitempty"`
	MinOccurs string `xml:"minOccurs,attr,omitempty"`
	MaxOccurs string `xml:"maxOccurs,attr,omitempty"`
}

// Attributes returns the attribute list for the named complex type
// — the typical WFS shape walks `complexType > complexContent >
// extension > sequence > element*`. Returns nil if the type is not
// present in the schema.
//
// typeName is the local name (without namespace prefix); pass an
// empty string to use the first complex type in the schema (the
// common case where DescribeFeatureType returns one type).
func (s *FeatureSchema) Attributes(typeName string) []Attribute {
	if s == nil || len(s.ComplexTypes) == 0 {
		return nil
	}
	var ct *ComplexType
	if typeName == "" {
		ct = &s.ComplexTypes[0]
	} else {
		for i := range s.ComplexTypes {
			if s.ComplexTypes[i].Name == typeName {
				ct = &s.ComplexTypes[i]
				break
			}
		}
	}
	if ct == nil {
		return nil
	}
	switch {
	case ct.Sequence != nil:
		return ct.Sequence.Elements
	case ct.ComplexContent != nil:
		return ct.ComplexContent.Extension.Sequence.Elements
	default:
		return nil
	}
}
