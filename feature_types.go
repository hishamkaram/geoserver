package geoserver

import (
	"fmt"
	"strconv"
)

// FeatureTypeService define all geoserver featuretype operations
type FeatureTypeService interface {
	GetFeatureTypes(workspaceName string, datastoreName string) (featureTypes []*Resource, err error)
	GetFeatureType(workspaceName string, datastoreName string, featureTypeName string) (featureType *FeatureType, err error)
	DeleteFeatureType(workspaceName string, datastoreName string, featureTypeName string, recurse bool) (deleted bool, err error)
}

// Entry is geoserver Entry
type Entry struct {
	Key   string `json:"@key,omitempty"`
	Value string `json:"$,omitempty"`
}

// BoundingBox is geoserver Bounding Box for FeatureType
type BoundingBox struct {
	Minx float64 `json:"minx,omitempty"`
	Maxx float64 `json:"maxx,omitempty"`
	Miny float64 `json:"miny,omitempty"`
	Maxy float64 `json:"maxy,omitempty"`
}

//Metadata is the geoserver Metadata
type Metadata struct {
	Entry []*Entry `json:"entry,omitempty"`
}

//Keywords is the geoserver Keywords
type Keywords struct {
	String []string `json:"string,omitempty"`
}

//ResponseSRS is the geoserver ResponseSRS
type ResponseSRS struct {
	String []int `json:"string,omitempty"`
}

// NativeBoundingBox is geoserver NativeBoundingBox for FeatureType
type NativeBoundingBox struct {
	BoundingBox
	Crs *interface{} `json:"crs,omitempty"`
}

// LatLonBoundingBox is geoserver LatLonBoundingBox for FeatureType
type LatLonBoundingBox struct {
	BoundingBox
	Crs string `json:"crs,omitempty"`
}

// MetadataLink is geoserver metadata link
type MetadataLink struct {
	Type         string `json:"type,omitempty"`
	MetadataType string `json:"metadataType,omitempty"`
	Content      string `json:"content,omitempty"`
}

//MetadataLinks is the geoserver metadata links
type MetadataLinks struct {
	MetadataLink []*MetadataLink `json:"metadataLink,omitempty"`
}

//DataLinks is the geoserver FeatureType Datalinks
type DataLinks struct {
	DataLink []*MetadataLink `json:"org.geoserver.catalog.impl.DataLinkInfoImpl,omitempty"`
}

//Attributes is the geoserver feature type attributes
type Attributes struct {
	Attribute []*Attribute `json:"attribute,omitempty"`
}

//NativeCRSAsEntry get CRS to Entry
func NativeCRSAsEntry(in interface{}) []Entry {
	nativeCRS := make([]Entry, 0)
	switch v := in.(type) {
	case map[string]interface{}:
		for k, value := range v {
			nativeCRS = append(nativeCRS, Entry{Key: k, Value: value.(string)})
		}

	case string:
		nativeCRS = append(nativeCRS, Entry{Key: "crs", Value: in.(string)})
	default:
		nativeCRS = append(nativeCRS, Entry{})
	}
	return nativeCRS
}

// Attribute is geoserver FeatureType Attribute
type Attribute struct {
	Name      string `json:"name,omitempty"`
	MinOccurs int16  `json:"minOccurs,omitempty"`
	MaxOccurs int16  `json:"maxOccurs,omitempty"`
	Nillable  bool   `json:"nillable,omitempty"`
	Binding   string `json:"binding,omitempty"`
	Length    int16  `json:"length,omitempty"`
}

// FeatureType is geoserver FeatureType
type FeatureType struct {
	Name                   string             `json:"name,omitempty"`
	NativeName             string             `json:"nativeName,omitempty"`
	Namespace              *Resource          `json:"namespace,omitempty"`
	Title                  string             `json:"title,omitempty"`
	Abstract               string             `json:"abstract,omitempty"`
	Keywords               *Keywords          `json:"keywords,omitempty"`
	Metadatalinks          *MetadataLinks     `json:"metadatalinks,omitempty"`
	DataLinks              *DataLinks         `json:"dataLinks,omitempty"`
	NativeCRS              *interface{}       `json:"nativeCRS,omitempty"`
	Srs                    string             `json:"srs,omitempty"`
	Enabled                bool               `json:"enabled,omitempty"`
	NativeBoundingBox      *NativeBoundingBox `json:"nativeBoundingBox,omitempty"`
	LatLonBoundingBox      *LatLonBoundingBox `json:"latLonBoundingBox,omitempty"`
	ProjectionPolicy       string             `json:"projectionPolicy,omitempty"`
	Metadata               *Metadata          `json:"metadata,omitempty"`
	Store                  *Resource          `json:"store,omitempty"`
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

// FeatureTypes holds a list of geoserver styles
type FeatureTypes struct {
	FeatureType []*Resource `json:"featureType,omitempty"`
}

//FeatureTypesResponseBody is the api body
type FeatureTypesResponseBody struct {
	FeatureTypes *FeatureTypes `json:"featureTypes,omitempty"`
}

//FeatureTypesRequestBody is the api body
type FeatureTypesRequestBody struct {
	FeatureType *FeatureType `json:"featureTypes,omitempty"`
}

// GetFeatureTypes return all featureTypes in workspace and datastore if error occurred err will be return and nil for featrueTypes
func (g *GeoServer) GetFeatureTypes(workspaceName string, datastoreName string) (featureTypes []*Resource, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	if datastoreName != "" {
		datastoreName = fmt.Sprintf("datastores/%s/featuretypes", datastoreName)
	}
	targetURL := g.ParseURL("rest", workspaceName, datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		featureTypes = nil
		err = g.GetError(responseCode, response)
		return
	}
	featureTypesResponse := &FeatureTypesResponseBody{FeatureTypes: &FeatureTypes{FeatureType: make([]*Resource, 0, 0)}}
	g.DeSerializeJSON(response, featureTypesResponse)
	featureTypes = featureTypesResponse.FeatureTypes.FeatureType
	return
}

// DeleteFeatureType Delete FeatureType from geoserver given that workspaceName, datastoreName, featureTypeName
// if featuretype deleted successfully will return true and nil for err
// if error occurred will return false and error for err
func (g *GeoServer) DeleteFeatureType(workspaceName string, datastoreName string, featureTypeName string, recurse bool) (deleted bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	if datastoreName != "" {
		datastoreName = fmt.Sprintf("datastores/%s/", datastoreName)
	}
	targetURL := g.ParseURL("rest", workspaceName, datastoreName, "featuretypes", featureTypeName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}

// GetFeatureType it return geoserver FeatureType and nil err
// if success else nil for fetureType error for err
func (g *GeoServer) GetFeatureType(workspaceName string, datastoreName string, featureTypeName string) (featureType *FeatureType, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	if datastoreName != "" {
		datastoreName = fmt.Sprintf("datastores/%s/featuretypes", datastoreName)
	}
	targetURL := g.ParseURL("rest", workspaceName, datastoreName, featureTypeName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		featureType = nil
		err = g.GetError(responseCode, response)
		return
	}
	var featureTypeResponse struct {
		FeatureType *FeatureType `json:"featureType,omitempty"`
	}
	g.DeSerializeJSON(response, &featureTypeResponse)
	featureType = featureTypeResponse.FeatureType
	return
}
