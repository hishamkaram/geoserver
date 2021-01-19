package geoserver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

// LayerService define  geoserver layers operations
type LayerService interface {

	//GetLayers  get all layers from workspace in geoserver else return error
	GetLayers(workspaceName string) (layers []*Resource, err error)

	// GetshpFiledsName datastore name from shapefile name
	GetshpFiledsName(filename string) string

	// UploadShapeFile upload shapefile to geoserver
	UploadShapeFile(fileURI string, workspaceName string, datastoreName string) (uploaded bool, err error)

	//GetLayer get specific Layer from geoserver else return error
	GetLayer(workspaceName string, layerName string) (layer *Layer, err error)

	//UpdateLayer partial update geoserver layer else return error
	UpdateLayer(workspaceName string, layerName string, layer Layer) (modified bool, err error)

	//DeleteLayer delete geoserver layer and its reources else return error
	DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, err error)

	PublishPostgisLayer(workspaceName string, datastoreName string, publishName string, tableName string) (published bool, err error)
}

//Resource geoserver resource
type Resource struct {
	Class string `json:"@class,omitempty"`
	Name  string `json:"name,omitempty"`
	Href  string `json:"href,omitempty"`
}

//Attribution of resource
type Attribution struct {
	Title      string `json:"title,omitempty"`
	Href       string `json:"href,omitempty"`
	LogoURL    string `json:"logoURL,omitempty"`
	LogoType   string `json:"logoType,omitempty"`
	LogoWidth  int    `json:"logoWidth,omitempty"`
	LogoHeight int    `json:"logoHeight,omitempty"`
}

//Layer geoserver layers
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

//LayerRequestBody api json
type LayerRequestBody struct {
	Layer Layer `json:"layer,omitempty"`
}

//PublishPostgisLayerRequest is the api body
type PublishPostgisLayerRequest struct {
	FeatureType *FeatureType `json:"featureType,omitempty"`
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
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		// g.logger.Error(err)
		return
	}

	exists, _ := g.WorkspaceExists(workspaceName)
	if !exists {
		g.CreateWorkspace(workspaceName)
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
		//g.logger.Error(string(response))
		uploaded = false
		err = g.GetError(responseCode, response)
		return
	}
	uploaded = true
	return

}

//GetLayers  get all layers from workspace in geoserver else return error,
//if workspace is "" the it will return all public layers in geoserver
func (g *GeoServer) GetLayers(workspaceName string) (layers []*Resource, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "layers")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		//g.logger.Error(string(response))
		layers = nil
		err = g.GetError(responseCode, response)
		return
	}
	var layerResponse struct {
		Layers struct {
			Layer []*Resource `json:"layer,omitempty"`
		} `json:"layers,omitempty"`
	}
	g.DeSerializeJSON(response, &layerResponse)
	layers = layerResponse.Layers.Layer
	return
}

//GetLayer get specific Layer in a workspace from geoserver else return error,
//if workspace is "" the it will return geoserver public layer with ${layerName}
func (g *GeoServer) GetLayer(workspaceName string, layerName string) (layer *Layer, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "layers", layerName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		//g.logger.Error(string(response))
		layer = &Layer{}
		err = g.GetError(responseCode, response)
		return
	}
	var layerResponse struct {
		Layer *Layer `json:"layer,omitempty"`
	}
	g.DeSerializeJSON(response, &layerResponse)
	layer = layerResponse.Layer
	return
}

//UpdateLayer partial update geoserver layer else return error,
//if workspace is "" the it will update  public layer with name ${layerName} in geoserver
func (g *GeoServer) UpdateLayer(workspaceName string, layerName string, layer Layer) (modified bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "layers", layerName)
	data := LayerRequestBody{Layer: layer}

	serializedLayer, _ := g.SerializeStruct(data)
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
		//g.logger.Error(string(response))
		modified = false
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

//PublishPostgisLayer publish postgis table to geoserver
func (g *GeoServer) PublishPostgisLayer(workspaceName string, datastoreName string, publishName string, tableName string) (published bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "datastores", datastoreName, "/featuretypes")
	data := PublishPostgisLayerRequest{FeatureType: &FeatureType{Name: publishName,
		NativeName: tableName}}

	serializedLayer, _ := g.SerializeStruct(data)
	//g.logger.Errorf("%s", serializedLayer)
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
		//g.logger.Error(response)
		published = false
		err = g.GetError(responseCode, response)
		return
	}
	published = true
	return
}

//DeleteLayer delete geoserver layer and its reources else return error,
//if workspace is "" will delete public layer with name ${layerName} if exists
func (g *GeoServer) DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "layers", layerName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		//g.logger.Error(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
