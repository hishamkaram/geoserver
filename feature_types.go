package geoserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// CRSType geoserver crs response
type CRSType struct {
	Class string `json:"@class,omitempty"`
	Value string `json:"$,omitempty"`
}

// UnmarshalJSON custom deserialization for the GeoServer CRS response, which
// can be either a JSON object ({"@class":"...","$":"..."}) or a bare string.
//
// In v1.0.x, type assertions on missing keys panicked; v1.1+ uses ok-checked
// assertions and reports a clear error if the payload shape is unexpected.
func (u *CRSType) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch raw := raw.(type) {
	case map[string]interface{}:
		class, _ := raw["@class"].(string)
		value, _ := raw["$"].(string)
		if class == "" && value == "" {
			return fmt.Errorf("feature_types: unrecognized CRS payload: %v", raw)
		}
		*u = CRSType{Class: class, Value: value}
	default:
		// Treat any non-object shape as a bare CRS string identifier.
		*u = CRSType{Class: "string", Value: string(data)}
	}
	return nil
}

// MarshalJSON custom crs serialization
func (u *CRSType) MarshalJSON() ([]byte, error) {
	if IsEmpty(u) {
		x := ""
		return json.Marshal(&x)
	} else if !IsEmpty(u.Class) && u.Class == "string" {
		return json.Marshal(u.Value)
	}
	type crsType struct {
		Class string `json:"@class,omitempty"`
		Value string `json:"$,omitempty"`
	}
	return json.Marshal(&crsType{
		Class: u.Class,
		Value: u.Value,
	})
}

// FeatureTypeService define all geoserver featuretype operations
type FeatureTypeService interface {
	GetFeatureTypes(workspaceName string, datastoreName string) (featureTypes []*Resource, err error)

	// GetFeatureTypeList lists feature-type names in a datastore filtered
	// by `kind`: "configured" (default), "available", "available_with_geom",
	// or "all". The "available" variants are useful for discovering tables
	// in the underlying datastore that have not yet been published as
	// GeoServer feature types.
	GetFeatureTypeList(workspaceName string, datastoreName string, kind FeatureTypeListKind) (names []string, err error)

	GetFeatureType(workspaceName string, datastoreName string, featureTypeName string) (featureType *FeatureType, err error)

	// CreateFeatureType creates a featureType in workspace and datastore.
	// Only valid for database-backed datastores (PostGIS, etc.) — for
	// shapefile/geopackage stores use UploadShapeFile or PublishPostgisLayer.
	CreateFeatureType(workspaceName string, datastoreName string, featureType *FeatureType) (created bool, err error)

	DeleteFeatureType(workspaceName string, datastoreName string, featureTypeName string, recurse bool) (deleted bool, err error)
}

// FeatureTypeServiceWithContext is the context-aware sibling of [FeatureTypeService].
type FeatureTypeServiceWithContext interface {
	GetFeatureTypesContext(ctx context.Context, workspaceName string, datastoreName string) (featureTypes []*Resource, err error)
	GetFeatureTypeListContext(ctx context.Context, workspaceName string, datastoreName string, kind FeatureTypeListKind) (names []string, err error)
	GetFeatureTypeContext(ctx context.Context, workspaceName string, datastoreName string, featureTypeName string) (featureType *FeatureType, err error)
	CreateFeatureTypeContext(ctx context.Context, workspaceName string, datastoreName string, featureType *FeatureType) (created bool, err error)
	DeleteFeatureTypeContext(ctx context.Context, workspaceName string, datastoreName string, featureTypeName string, recurse bool) (deleted bool, err error)
}

// FeatureTypeListKind is the GeoServer `?list=` query value for the feature
// types listing endpoint. It controls which subset of tables / configured
// types is returned.
type FeatureTypeListKind string

// Recognized FeatureTypeListKind values. GeoServer's REST API:
//   - configured (default): only feature types that already have a GeoServer config
//   - available:            tables in the datastore not yet configured
//   - available_with_geom:  same as available, but only tables with a geometry column
//   - all:                  configured ∪ available
const (
	FeatureTypeListConfigured        FeatureTypeListKind = "configured"
	FeatureTypeListAvailable         FeatureTypeListKind = "available"
	FeatureTypeListAvailableWithGeom FeatureTypeListKind = "available_with_geom"
	FeatureTypeListAll               FeatureTypeListKind = "all"
)

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

// Metadata is the geoserver Metadata
type Metadata struct {
	Entry []*Entry `json:"entry,omitempty"`
}

// Keywords is the geoserver Keywords
type Keywords struct {
	String []string `json:"string,omitempty"`
}

// ResponseSRS is the geoserver ResponseSRS
type ResponseSRS struct {
	String []int `json:"string,omitempty"`
}

// NativeBoundingBox is geoserver NativeBoundingBox for FeatureType
type NativeBoundingBox struct {
	BoundingBox
	Crs *CRSType `json:"crs,omitempty"`
}

// LatLonBoundingBox is geoserver LatLonBoundingBox for FeatureType
type LatLonBoundingBox struct {
	BoundingBox
	Crs *CRSType `json:"crs,omitempty"`
}

// MetadataLink is geoserver metadata link
type MetadataLink struct {
	Type         string `json:"type,omitempty"`
	MetadataType string `json:"metadataType,omitempty"`
	Content      string `json:"content,omitempty"`
}

// MetadataLinks is the geoserver metadata links
type MetadataLinks struct {
	MetadataLink []*MetadataLink `json:"metadataLink,omitempty"`
}

// DataLinks is the geoserver FeatureType Datalinks
type DataLinks struct {
	DataLink []*MetadataLink `json:"org.geoserver.catalog.impl.DataLinkInfoImpl,omitempty"`
}

// Attributes is the geoserver feature type attributes
type Attributes struct {
	Attribute []*Attribute `json:"attribute,omitempty"`
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
	NativeCRS              *CRSType           `json:"nativeCRS,omitempty"`
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

// FeatureTypesResponseBody is the api body
type FeatureTypesResponseBody struct {
	FeatureTypes *FeatureTypes `json:"featureTypes,omitempty"`
}

// FeatureTypesRequestBody is the api body
type FeatureTypesRequestBody struct {
	FeatureType *FeatureType `json:"featureTypes,omitempty"`
}

// featureTypesURL builds /rest[/workspaces/{ws}/datastores/{ds}]/featuretypes[/{name}]
// with proper escaping. workspaceName and datastoreName may be empty
// (returns the global endpoint), but if datastoreName is provided then
// workspaceName must also be provided.
func (g *GeoServer) featureTypesURL(workspaceName, datastoreName string, extra ...string) string {
	parts := []string{"rest"}
	if workspaceName != "" {
		parts = append(parts, "workspaces", workspaceName)
		if datastoreName != "" {
			parts = append(parts, "datastores", datastoreName)
		}
	}
	parts = append(parts, "featuretypes")
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// featureTypeListResponse models GeoServer's `?list=...` response shape:
//
//	{"list":{"string":["table1","table2"]}}
type featureTypeListResponse struct {
	List struct {
		Strings []string `json:"string"`
	} `json:"list"`
}

// GetFeatureTypeList lists feature-type names in a datastore using context.Background.
// See [GeoServer.GetFeatureTypeListContext].
func (g *GeoServer) GetFeatureTypeList(workspaceName string, datastoreName string, kind FeatureTypeListKind) (names []string, err error) {
	return g.GetFeatureTypeListContext(context.Background(), workspaceName, datastoreName, kind)
}

// GetFeatureTypeListContext is the context-aware variant of [GeoServer.GetFeatureTypeList].
//
// kind controls the GeoServer `?list=` query parameter; if empty it defaults
// to [FeatureTypeListAll].
func (g *GeoServer) GetFeatureTypeListContext(ctx context.Context, workspaceName string, datastoreName string, kind FeatureTypeListKind) (names []string, err error) {
	if kind == "" {
		kind = FeatureTypeListAll
	}
	targetURL := g.featureTypesURL(workspaceName, datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"list": string(kind)},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		return nil, g.GetError(responseCode, response)
	}
	var body featureTypeListResponse
	if err = g.DeSerializeJSON(response, &body); err != nil {
		return nil, fmt.Errorf("GetFeatureTypeList: decode: %w", err)
	}
	return body.List.Strings, nil
}

// GetFeatureTypes lists feature types using context.Background.
func (g *GeoServer) GetFeatureTypes(workspaceName string, datastoreName string) (featureTypes []*Resource, err error) {
	return g.GetFeatureTypesContext(context.Background(), workspaceName, datastoreName)
}

// GetFeatureTypesContext is the context-aware variant of [GeoServer.GetFeatureTypes].
func (g *GeoServer) GetFeatureTypesContext(ctx context.Context, workspaceName string, datastoreName string) (featureTypes []*Resource, err error) {
	targetURL := g.featureTypesURL(workspaceName, datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		featureTypes = nil
		err = g.GetError(responseCode, response)
		return
	}
	featureTypesResponse := &FeatureTypesResponseBody{FeatureTypes: &FeatureTypes{FeatureType: make([]*Resource, 0)}}
	if err = g.DeSerializeJSON(response, featureTypesResponse); err != nil {
		return nil, err
	}
	featureTypes = featureTypesResponse.FeatureTypes.FeatureType
	return
}

// CreateFeatureType creates a featureType in workspace and datastore using context.Background.
//
// Only valid for database-backed datastores (PostGIS, Oracle, etc.). For
// shapefile-based stores, prefer [GeoServer.UploadShapeFile] which creates
// the underlying datastore and feature type in one step.
func (g *GeoServer) CreateFeatureType(workspaceName string, datastoreName string, featureType *FeatureType) (created bool, err error) {
	return g.CreateFeatureTypeContext(context.Background(), workspaceName, datastoreName, featureType)
}

// CreateFeatureTypeContext is the context-aware variant of [GeoServer.CreateFeatureType].
func (g *GeoServer) CreateFeatureTypeContext(ctx context.Context, workspaceName string, datastoreName string, featureType *FeatureType) (created bool, err error) {
	targetURL := g.featureTypesURL(workspaceName, datastoreName)
	body := struct {
		FeatureType *FeatureType `json:"featureType"`
	}{featureType}
	data, serErr := g.SerializeStruct(body)
	if serErr != nil {
		return false, fmt.Errorf("CreateFeatureType: serialize feature type: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(data),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteFeatureType deletes a feature type using context.Background.
func (g *GeoServer) DeleteFeatureType(workspaceName string, datastoreName string, featureTypeName string, recurse bool) (deleted bool, err error) {
	return g.DeleteFeatureTypeContext(context.Background(), workspaceName, datastoreName, featureTypeName, recurse)
}

// DeleteFeatureTypeContext is the context-aware variant of [GeoServer.DeleteFeatureType].
func (g *GeoServer) DeleteFeatureTypeContext(ctx context.Context, workspaceName string, datastoreName string, featureTypeName string, recurse bool) (deleted bool, err error) {
	targetURL := g.featureTypesURL(workspaceName, datastoreName, featureTypeName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}

// GetFeatureType fetches a feature type using context.Background.
func (g *GeoServer) GetFeatureType(workspaceName string, datastoreName string, featureTypeName string) (featureType *FeatureType, err error) {
	return g.GetFeatureTypeContext(context.Background(), workspaceName, datastoreName, featureTypeName)
}

// GetFeatureTypeContext is the context-aware variant of [GeoServer.GetFeatureType].
func (g *GeoServer) GetFeatureTypeContext(ctx context.Context, workspaceName string, datastoreName string, featureTypeName string) (featureType *FeatureType, err error) {
	targetURL := g.featureTypesURL(workspaceName, datastoreName, featureTypeName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		featureType = nil
		err = g.GetError(responseCode, response)
		return
	}
	var featureTypeResponse struct {
		FeatureType *FeatureType `json:"featureType,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &featureTypeResponse); err != nil {
		return nil, err
	}
	featureType = featureTypeResponse.FeatureType
	return
}
