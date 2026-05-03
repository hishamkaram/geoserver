package geoserver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LayerService define  geoserver layers operations
type LayerService interface {

	// GetLayers  get all layers from workspace in geoserver else return error
	GetLayers(workspaceName string) (layers []*Resource, err error)

	// GetshpFiledsName datastore name from shapefile name
	GetshpFiledsName(filename string) string

	// UploadShapeFile upload shapefile to geoserver
	UploadShapeFile(fileURI string, workspaceName string, datastoreName string) (uploaded bool, err error)

	// GetLayer get specific Layer from geoserver else return error
	GetLayer(workspaceName string, layerName string) (layer *Layer, err error)

	// UpdateLayer partial update geoserver layer else return error
	UpdateLayer(workspaceName string, layerName string, layer Layer) (modified bool, err error)

	// DeleteLayer delete geoserver layer and its reources else return error
	DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, err error)

	PublishPostgisLayer(workspaceName string, datastoreName string, publishName string, tableName string) (published bool, err error)

	PublishGeoTiffLayer(workspaceName string, coveragestoreName string, publishName string, fileName string) (published bool, err error)
}

// Resource geoserver resource
type Resource struct {
	Class string `json:"@class,omitempty"`
	Name  string `json:"name,omitempty"`
	Href  string `json:"href,omitempty"`
}

// Attribution of resource
type Attribution struct {
	Title      string `json:"title,omitempty"`
	Href       string `json:"href,omitempty"`
	LogoURL    string `json:"logoURL,omitempty"`
	LogoType   string `json:"logoType,omitempty"`
	LogoWidth  int    `json:"logoWidth,omitempty"`
	LogoHeight int    `json:"logoHeight,omitempty"`
}

// Layer geoserver layers
type Layer struct {
	Name         string    `json:"name,omitempty"`
	Path         string    `json:"path,omitempty"`
	Type         string    `json:"type,omitempty"`
	DefaultStyle *Resource `json:"defaultStyle,omitempty"`
	Styles       *struct {
		Class string     `json:"@class,omitempty"`
		Style []Resource `json:"style,omitempty"`
	} `json:"styles,omitempty"`
	Resource    Resource     `json:"resource,omitempty"`
	Queryable   bool         `json:"queryable,omitempty"`
	Opaque      bool         `json:"opaque,omitempty"`
	Attribution *Attribution `json:"attribution,omitempty"`
}

// LayerRequestBody api json
type LayerRequestBody struct {
	Layer Layer `json:"layer,omitempty"`
}

// PublishPostgisLayerRequest is the api body
type PublishPostgisLayerRequest struct {
	FeatureType *FeatureType `json:"featureType,omitempty"`
}

// layersURL builds /rest[/workspaces/{ws}]/layers[/{name}] with proper escaping.
func (g *GeoServer) layersURL(workspaceName string, extra ...string) string {
	parts := []string{"rest"}
	if workspaceName != "" {
		parts = append(parts, "workspaces", workspaceName)
	}
	parts = append(parts, "layers")
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// GetshpFiledsName datastore name from shapefile name
func (g *GeoServer) GetshpFiledsName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name
}

// UploadShapeFile upload shapefile to geoserver
func (g *GeoServer) UploadShapeFile(fileURI string, workspaceName string, datastoreName string) (uploaded bool, err error) {
	filename := filepath.Base(fileURI)
	if datastoreName == "" {
		datastoreName = g.GetshpFiledsName(filename)
	}
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName, "file.shp")
	shapeFileBinary, err := os.ReadFile(fileURI)
	if err != nil {
		g.logger.Error(err)
		return
	}

	exists, existsErr := g.WorkspaceExists(workspaceName)
	if existsErr != nil {
		// Don't fail outright — proceed to attempt the upload, which will
		// surface the underlying error with full context if it persists.
		g.logger.Warnf("UploadShapeFile: WorkspaceExists(%q) returned %v; attempting upload anyway", workspaceName, existsErr)
	}
	if !exists {
		if _, createErr := g.CreateWorkspace(workspaceName); createErr != nil {
			return false, fmt.Errorf("UploadShapeFile: create workspace %q: %w", workspaceName, createErr)
		}
	}
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(shapeFileBinary),
		DataType: zipType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		uploaded = false
		err = g.GetError(responseCode, response)
		return
	}
	uploaded = true
	return
}

// GetLayers  get all layers from workspace in geoserver else return error,
// if workspace is "" the it will return all public layers in geoserver
func (g *GeoServer) GetLayers(workspaceName string) (layers []*Resource, err error) {
	targetURL := g.layersURL(workspaceName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		layers = nil
		err = g.GetError(responseCode, response)
		return
	}
	var layerResponse struct {
		Layers struct {
			Layer []*Resource `json:"layer,omitempty"`
		} `json:"layers,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &layerResponse); err != nil {
		return nil, err
	}
	layers = layerResponse.Layers.Layer
	return
}

// GetLayer get specific Layer in a workspace from geoserver else return error,
// if workspace is "" the it will return geoserver public layer with ${layerName}
func (g *GeoServer) GetLayer(workspaceName string, layerName string) (layer *Layer, err error) {
	targetURL := g.layersURL(workspaceName, layerName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		layer = &Layer{}
		err = g.GetError(responseCode, response)
		return
	}
	var layerResponse struct {
		Layer *Layer `json:"layer,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &layerResponse); err != nil {
		return nil, err
	}
	layer = layerResponse.Layer
	return
}

// UpdateLayer partial update geoserver layer else return error,
// if workspace is "" the it will update  public layer with name ${layerName} in geoserver
func (g *GeoServer) UpdateLayer(workspaceName string, layerName string, layer Layer) (modified bool, err error) {
	targetURL := g.layersURL(workspaceName, layerName)
	data := LayerRequestBody{Layer: layer}

	serializedLayer, serErr := g.SerializeStruct(data)
	if serErr != nil {
		return false, fmt.Errorf("UpdateLayer: serialize layer: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		modified = false
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

// PublishPostgisLayer publish postgis table to geoserver
func (g *GeoServer) PublishPostgisLayer(workspaceName string, datastoreName string, publishName string, tableName string) (published bool, err error) {
	parts := []string{"rest"}
	if workspaceName != "" {
		parts = append(parts, "workspaces", workspaceName)
	}
	parts = append(parts, "datastores", datastoreName, "featuretypes")
	targetURL := g.ParseURL(parts...)
	data := PublishPostgisLayerRequest{FeatureType: &FeatureType{
		Name:       publishName,
		NativeName: tableName,
	}}

	serializedLayer, serErr := g.SerializeStruct(data)
	if serErr != nil {
		return false, fmt.Errorf("PublishPostgisLayer: serialize feature type: %w", serErr)
	}
	g.logger.Debugf("PublishPostgisLayer: body=%s", serializedLayer)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		published = false
		err = g.GetError(responseCode, response)
		return
	}
	published = true
	return
}

// DeleteLayer delete geoserver layer and its reources else return error,
// if workspace is "" will delete public layer with name ${layerName} if exists
func (g *GeoServer) DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, err error) {
	targetURL := g.layersURL(workspaceName, layerName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
