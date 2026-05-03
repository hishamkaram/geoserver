// Package wcs is the v2 sub-client for the GeoServer WCS service.
// Covers the GetCapabilities endpoint — fetching the XML capabilities
// document and parsing it into Go types.
//
// The GeoServer WCS GetCapabilities response uses both `wcs:` and
// `ows:` XML namespaces; type definitions match on local name only,
// so values flow through Go's encoding/xml decoder regardless of
// which namespace prefix the server applied. The default version
// supported by this type tree is WCS 2.0.1 — GeoServer's modern
// default. WCS 1.0.0 / 1.1.1 use a different root element
// (`WCS_Capabilities`) and are not in scope here.
package wcs

import "encoding/xml"

// Capabilities is the root of the WCS 2.0.x GetCapabilities document.
// GeoServer emits `<wcs:Capabilities>` for 2.0.x; the XMLName matches
// by local name so namespace differences don't trip the decoder.
type Capabilities struct {
	XMLName               xml.Name              `xml:"Capabilities"`
	Version               string                `xml:"version,attr,omitempty"`
	UpdateSequence        string                `xml:"updateSequence,attr,omitempty"`
	ServiceIdentification ServiceIdentification `xml:"ServiceIdentification"`
	ServiceProvider       ServiceProvider       `xml:"ServiceProvider"`
	OperationsMetadata    OperationsMetadata    `xml:"OperationsMetadata"`
	ServiceMetadata       ServiceMetadata       `xml:"ServiceMetadata"`
	Contents              Contents              `xml:"Contents"`
}

// ServiceIdentification carries the service-level metadata block.
type ServiceIdentification struct {
	Title             string   `xml:"Title"`
	Abstract          string   `xml:"Abstract"`
	Keywords          []string `xml:"Keywords>Keyword"`
	ServiceType       string   `xml:"ServiceType"`
	Versions          []string `xml:"ServiceTypeVersion"`
	Profiles          []string `xml:"Profile"`
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
// any linkable element.
type OnlineResource struct {
	Type string `xml:"http://www.w3.org/1999/xlink type,attr,omitempty"`
	Href string `xml:"http://www.w3.org/1999/xlink href,attr,omitempty"`
}

// OperationsMetadata enumerates the WCS operations the server
// advertises along with each operation's DCP endpoints, parameters,
// and constraints.
type OperationsMetadata struct {
	Operation []Operation `xml:"Operation"`
}

// Operation describes one server operation (GetCapabilities,
// DescribeCoverage, GetCoverage).
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
// allowed values.
type Parameter struct {
	Name          string   `xml:"name,attr"`
	AllowedValues []string `xml:"AllowedValues>Value"`
	DefaultValue  string   `xml:"DefaultValue"`
}

// ServiceMetadata advertises encoding formats and supported CRSes.
// The wire shape uses the `wcs:` namespace; structure carries forward
// the standard GeoServer fields.
type ServiceMetadata struct {
	Formats []string `xml:"formatSupported"`
	CRS     []string `xml:"Extension>crsSupported"`
}

// Contents enumerates the published coverages.
type Contents struct {
	CoverageSummary []CoverageSummary `xml:"CoverageSummary"`
}

// CoverageSummary is one published coverage entry. WCS 2.0 minimizes
// the per-coverage envelope to CoverageId + CoverageSubtype; richer
// per-coverage detail is fetched via DescribeCoverage (not in this
// package's scope).
type CoverageSummary struct {
	CoverageID      string `xml:"CoverageId"`
	CoverageSubtype string `xml:"CoverageSubtype"`
}
