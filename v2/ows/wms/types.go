// Package wms is the v2 sub-client for the GeoServer WMS service. It
// covers the GetCapabilities endpoint — fetching the XML capabilities
// document and parsing it into Go types. The exported XML types mirror
// v1's wms package one-for-one so callers can move with no shape
// changes; the parser accepts io.Reader (v2 idiom) instead of []byte.
//
// WMS request endpoints (GetMap, GetFeatureInfo, GetLegendGraphic) are
// HTTP-content endpoints rather than catalog operations and stay on
// the application layer. WFS / WCS GetCapabilities can land alongside
// in this package's siblings (ows/wfs, ows/wcs) using the same shape.
package wms

import "encoding/xml"

// OnlineResource is the xlink:href + xlink:type pair returned for any
// linkable element (request DCP endpoints, MetadataURL, LegendURL, …).
type OnlineResource struct {
	XMLName xml.Name `xml:"OnlineResource"`
	Type    string   `xml:"http://www.w3.org/1999/xlink type,attr,omitempty"`
	Href    string   `xml:"http://www.w3.org/1999/xlink href,attr,omitempty"`
}

// KeywordList is the keyword list attached to a Service or Layer.
type KeywordList struct {
	XMLName xml.Name  `xml:"KeywordList,omitempty"`
	Keyword []*string `xml:"Keyword,omitempty"`
}

// Get is the GET DCP-type endpoint descriptor.
type Get struct {
	XMLName        xml.Name       `xml:"Get"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

// Post is the POST DCP-type endpoint descriptor.
type Post struct {
	XMLName        xml.Name       `xml:"Post"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

// HTTP wraps the GET / POST DCP-type endpoints.
type HTTP struct {
	XMLName xml.Name `xml:"HTTP"`
	Get     *Get     `xml:"Get,omitempty"`
	Post    *Post    `xml:"Post,omitempty"`
}

// DCPType wraps the HTTP transport descriptor for a request.
type DCPType struct {
	XMLName xml.Name `xml:"DCPType,omitempty"`
	HTTP    HTTP     `xml:"HTTP,omitempty"`
}

// RequestEntry describes a single request — its supported Format list
// and DCPType endpoints. Used inside [Request].
type RequestEntry struct {
	XMLName xml.Name
	Format  []*string `xml:"Format,omitempty"`
	DCPType DCPType   `xml:"DCPType,omitempty"`
}

// Request enumerates the WMS operations the server supports along
// with each operation's DCP endpoints and supported formats.
type Request struct {
	XMLName          xml.Name     `xml:"Request"`
	GetCapabilities  RequestEntry `xml:"GetCapabilities"`
	GetMap           RequestEntry `xml:"GetMap"`
	GetFeatureInfo   RequestEntry `xml:"GetFeatureInfo"`
	DescribeLayer    RequestEntry `xml:"DescribeLayer"`
	GetLegendGraphic RequestEntry `xml:"GetLegendGraphic"`
	GetStyles        RequestEntry `xml:"GetStyles"`
}

// UserDefinedSymbolization is the SLD UserDefinedSymbolization element
// telling clients whether SLD/UserStyle/UserLayer/RemoteWFS are
// supported.
type UserDefinedSymbolization struct {
	XMLName    xml.Name `xml:"UserDefinedSymbolization"`
	SupportSLD string   `xml:"SupportSLD,attr"`
	UserLayer  string   `xml:"UserLayer,attr"`
	UserStyle  string   `xml:"UserStyle,attr"`
	RemoteWFS  string   `xml:"RemoteWFS,attr"`
}

// Exception lists the supported exception report formats.
type Exception struct {
	XMLName xml.Name  `xml:"Exception"`
	Format  []*string `xml:"Format"`
}

// AuthorityURL points at the authority that issued an [Identifier]
// (CRS code, layer ID, …). Carries an OnlineResource for the
// authority's URL.
type AuthorityURL struct {
	XMLName        xml.Name       `xml:"AuthorityURL"`
	Name           string         `xml:"name,attr"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

// LatLonBoundingBox is the lat/lon extent of a Layer.
type LatLonBoundingBox struct {
	XMLName xml.Name `xml:"LatLonBoundingBox"`
	MinX    float64  `xml:"minx,attr"`
	MinY    float64  `xml:"miny,attr"`
	MaxX    float64  `xml:"maxx,attr"`
	MaxY    float64  `xml:"maxy,attr"`
}

// BoundingBox is a layer's CRS-specific extent.
type BoundingBox struct {
	XMLName xml.Name `xml:"BoundingBox"`
	MinX    float64  `xml:"minx,attr"`
	MinY    float64  `xml:"miny,attr"`
	MaxX    float64  `xml:"maxx,attr"`
	MaxY    float64  `xml:"maxy,attr"`
	SRS     string   `xml:"SRS,attr,omitempty"`
}

// LegendURL is a renderable legend for a Style.
type LegendURL struct {
	XMLName        xml.Name       `xml:"LegendURL"`
	Height         float64        `xml:"height,attr"`
	Width          float64        `xml:"width,attr"`
	Format         string         `xml:"Format"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

// Style describes one rendering style for a Layer.
type Style struct {
	XMLName   xml.Name  `xml:"Style"`
	Name      string    `xml:"Name"`
	Title     string    `xml:"Title"`
	LegendURL LegendURL `xml:"LegendURL"`
}

// MetadataURL is a layer-level pointer to an external metadata document.
type MetadataURL struct {
	XMLName        xml.Name       `xml:"MetadataURL"`
	Type           string         `xml:"type,attr"`
	Format         string         `xml:"Format"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

// LogoURL is the logo image for a Service or Attribution.
type LogoURL struct {
	XMLName        xml.Name       `xml:"LogoURL"`
	Width          float32        `xml:"width,attr"`
	Height         float32        `xml:"height,attr"`
	Format         string         `xml:"Format"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

// Dimension declares a non-spatial dimension axis on a Layer
// (typically time, elevation, custom).
type Dimension struct {
	XMLName xml.Name `xml:"Dimension"`
	Name    string   `xml:"name,attr,omitempty"`
	Units   string   `xml:"units,attr,omitempty"`
}

// Extent declares the active value range on a [Dimension].
type Extent struct {
	XMLName xml.Name `xml:"Extent"`
	Name    string   `xml:"name,attr,omitempty"`
	Default string   `xml:"default,attr,omitempty"`
}

// Attribution is the per-Layer attribution block (title, online
// resource, logo).
type Attribution struct {
	XMLName        xml.Name       `xml:"Attribution"`
	Title          string         `xml:"Title,omitempty"`
	OnlineResource OnlineResource `xml:"OnlineResource,omitempty"`
	LogoURL        LogoURL        `xml:"LogoURL,omitempty"`
}

// Layer is one published WMS layer. Layers nest — `Layer.Layer` is
// the child list when this is a layer group / category.
type Layer struct {
	XMLName           xml.Name          `xml:"Layer"`
	Title             string            `xml:"Title"`
	Abstract          string            `xml:"Abstract"`
	Queryable         int8              `xml:"queryable,attr,omitempty"`
	SRS               []*string         `xml:"SRS,omitempty"`
	LatLonBoundingBox LatLonBoundingBox `xml:"LatLonBoundingBox,omitempty"`
	BoundingBox       []*BoundingBox    `xml:"BoundingBox,omitempty"`
	AuthorityURL      AuthorityURL      `xml:"AuthorityURL,omitempty"`
	Style             []*Style          `xml:"Style,omitempty"`
	Layer             []*Layer          `xml:"Layer,omitempty"`
	MetadataURL       []*MetadataURL    `xml:"MetadataURL,omitempty"`
	Dimension         Dimension         `xml:"Dimension,omitempty"`
	Extent            Extent            `xml:"Extent,omitempty"`
	Attribution       Attribution       `xml:"Attribution,omitempty"`
}

// Capability is the operations + advertised layer tree.
type Capability struct {
	XMLName                  xml.Name                 `xml:"Capability"`
	Request                  Request                  `xml:"Request"`
	Exception                Exception                `xml:"Exception"`
	UserDefinedSymbolization UserDefinedSymbolization `xml:"UserDefinedSymbolization"`
	Layer                    Layer                    `xml:"Layer"`
}

// Service is the service-level metadata block (title, keywords, fees,
// access constraints).
type Service struct {
	XMLName           xml.Name       `xml:"Service"`
	Name              string         `xml:"Name"`
	Title             string         `xml:"Title"`
	KeywordList       KeywordList    `xml:"KeywordList"`
	OnlineResource    OnlineResource `xml:"OnlineResource"`
	Fees              string         `xml:"Fees"`
	AccessConstraints string         `xml:"AccessConstraints"`
}

// Capabilities is the root of the WMS GetCapabilities document
// (`<WMT_MS_Capabilities>` for WMS 1.1.1).
type Capabilities struct {
	XMLName        xml.Name   `xml:"WMT_MS_Capabilities"`
	Version        string     `xml:"version,attr,omitempty"`
	UpdateSequence string     `xml:"updateSequence,attr,omitempty"`
	Service        Service    `xml:"Service"`
	Capability     Capability `xml:"Capability"`
}
