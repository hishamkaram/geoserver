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
		g.logger.Error(err)
		return
	}

	exists, _ := g.WorkspaceExists(workspaceName)
	if !exists {
		g.CreateWorkspace(workspaceName)
	}
	response, responseCode := g.DoPut(targetURL, bytes.NewBuffer(shapeFileBinary), zipType, "")
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		uploaded = false
		err = statusErrorMapping[responseCode]
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
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		layers = nil
		err = statusErrorMapping[responseCode]
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
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		layer = &Layer{}
		err = statusErrorMapping[responseCode]
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
	response, responseCode := g.DoPut(targetURL, bytes.NewBuffer(serializedLayer), jsonType, jsonType)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		modified = false
		err = statusErrorMapping[responseCode]
		return
	}
	modified = true
	return
}

//DeleteLayer delete geoserver layer and its reources else return error,
//if workspace is "" will delete public layer with name ${layerName} if exists
func (g *GeoServer) DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "layers", layerName)
	response, responseCode := g.DoDelete(targetURL, jsonType, map[string]string{"recurse": strconv.FormatBool(recurse)})
	if responseCode != statusOk {
		g.logger.Error(string(response))
		deleted = false
		err = statusErrorMapping[responseCode]
		return
	}
	deleted = true
	return
}
