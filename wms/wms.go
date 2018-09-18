package wms

import (
	"encoding/xml"
	"log"
)

//OnlineResource tag
type OnlineResource struct {
	XMLName xml.Name `xml:"OnlineResource"`
	Type    string   `xml:"http://www.w3.org/1999/xlink type,attr,omitempty"`
	Href    string   `xml:"http://www.w3.org/1999/xlink href,attr,omitempty"`
}

//KeywordList tag
type KeywordList struct {
	XMLName xml.Name  `xml:"KeywordList,omitempty"`
	Keyword []*string `xml:"Keyword,omitempty"`
}

//Get tag
type Get struct {
	XMLName        xml.Name       `xml:"Get"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

//Post tag
type Post struct {
	XMLName        xml.Name       `xml:"Post"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

//HTTP tag
type HTTP struct {
	XMLName xml.Name `xml:"HTTP"`
	Get     *Get     `xml:"Get,omitempty"`
	Post    *Post    `xml:"Post,omitempty"`
}

//DCPType tag
type DCPType struct {
	XMLName xml.Name `xml:"DCPType,omitempty"`
	HTTP    HTTP     `xml:"HTTP,omitempty"`
}

//RequestEntry tag
type RequestEntry struct {
	XMLName xml.Name
	Format  []*string `xml:"Format,omitempty"`
	DCPType DCPType   `xml:"DCPType,omitempty"`
}

//Request tag
type Request struct {
	XMLName          xml.Name     `xml:"Request"`
	GetCapabilities  RequestEntry `xml:"GetCapabilities"`
	GetMap           RequestEntry `xml:"GetMap"`
	GetFeatureInfo   RequestEntry `xml:"GetFeatureInfo"`
	DescribeLayer    RequestEntry `xml:"DescribeLayer"`
	GetLegendGraphic RequestEntry `xml:"GetLegendGraphic"`
	GetStyles        RequestEntry `xml:"GetStyles"`
}

//UserDefinedSymbolization tag
type UserDefinedSymbolization struct {
	XMLName    xml.Name `xml:"UserDefinedSymbolization"`
	SupportSLD string   `xml:"SupportSLD,attr"`
	UserLayer  string   `xml:"UserLayer,attr"`
	UserStyle  string   `xml:"UserStyle,attr"`
	RemoteWFS  string   `xml:"RemoteWFS,attr"`
}

//Exception tag
type Exception struct {
	XMLName xml.Name  `xml:"Exception"`
	Format  []*string `xml:"Format"`
}

//AuthorityURL tag
type AuthorityURL struct {
	XMLName        xml.Name       `xml:"AuthorityURL"`
	Name           string         `xml:"name,attr"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

//LatLonBoundingBox tag
type LatLonBoundingBox struct {
	XMLName xml.Name `xml:"LatLonBoundingBox"`
	MinX    float64  `xml:"minx,attr"`
	MinY    float64  `xml:"miny,attr"`
	MaxX    float64  `xml:"maxx,attr"`
	MaxY    float64  `xml:"maxy,attr"`
}

//BoundingBox tag
type BoundingBox struct {
	XMLName xml.Name `xml:"BoundingBox"`
	MinX    float64  `xml:"minx,attr"`
	MinY    float64  `xml:"miny,attr"`
	MaxX    float64  `xml:"maxx,attr"`
	MaxY    float64  `xml:"maxy,attr"`
	SRS     string   `xml:"SRS,attr,omitempty"`
}

//LegendURL tag
type LegendURL struct {
	XMLName        xml.Name       `xml:"LegendURL"`
	Height         float64        `xml:"height,attr"`
	Width          float64        `xml:"width,attr"`
	Format         string         `xml:"Format"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

//Style tag
type Style struct {
	XMLName   xml.Name  `xml:"Style"`
	Name      string    `xml:"Name"`
	Title     string    `xml:"Title"`
	LegendURL LegendURL `xml:"LegendURL"`
}

//MetadataURL tag
type MetadataURL struct {
	XMLName        xml.Name       `xml:"MetadataURL"`
	Type           string         `xml:"type,attr"`
	Format         string         `xml:"Format"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

//LogoURL tag
type LogoURL struct {
	XMLName        xml.Name       `xml:"LogoURL"`
	Width          float32        `xml:"width,attr"`
	Height         float32        `xml:"height,attr"`
	Format         string         `xml:"Format"`
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

//Dimension tag
type Dimension struct {
	XMLName xml.Name `xml:"Dimension"`
	Name    string   `xml:"name,attr,omitempty"`
	Units   string   `xml:"units,attr,omitempty"`
}

//Extent tag
type Extent struct {
	XMLName xml.Name `xml:"Extent"`
	Name    string   `xml:"name,attr,omitempty"`
	Default string   `xml:"default,attr,omitempty"`
}

//Attribution tag
type Attribution struct {
	XMLName        xml.Name       `xml:"Attribution"`
	Title          string         `xml:"Title,omitempty"`
	OnlineResource OnlineResource `xml:"OnlineResource,omitempty"`
	LogoURL        LogoURL        `xml:"LogoURL,omitempty"`
}

//Layer tag
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

//Capability tag
type Capability struct {
	XMLName                  xml.Name                 `xml:"Capability"`
	Request                  Request                  `xml:"Request"`
	Exception                Exception                `xml:"Exception"`
	UserDefinedSymbolization UserDefinedSymbolization `xml:"UserDefinedSymbolization"`
	Layer                    Layer                    `xml:"Layer"`
}

//Service tag
type Service struct {
	XMLName           xml.Name       `xml:"Service"`
	Name              string         `xml:"Name"`
	Title             string         `xml:"Title"`
	KeywordList       KeywordList    `xml:"KeywordList"`
	OnlineResource    OnlineResource `xml:"OnlineResource"`
	Fees              string         `xml:"Fees"`
	AccessConstraints string         `xml:"AccessConstraints"`
}

//Capabilities parent tag
type Capabilities struct {
	XMLName        xml.Name   `xml:"WMT_MS_Capabilities"`
	Version        string     `xml:"version,attr,omitempty"`
	UpdateSequence string     `xml:"updateSequence,attr,omitempty"`
	Service        Service    `xml:"Service"`
	Capability     Capability `xml:"Capability"`
}

//ParseCapabilities read wms capabilities
func ParseCapabilities(xmlByte []byte) *Capabilities {
	var cap Capabilities
	err := xml.Unmarshal(xmlByte, &cap)
	if err != nil {
		log.Fatal(err)
	}
	return &cap
}
