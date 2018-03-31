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
	//GetLayers  get all geoserver layers
	GetLayers(workspaceName string) (layers []Resource, statusCode int)
	// GetshpFiledsName datastore name from shapefile name
	GetShpdatastore(filename string) string

	// UploadShapeFile upload shapefile to geoserver
	UploadShapeFile(fileURI string, WorkspaceName string, datastoreName string) ([]byte, int)
	//GetLayer  get specific Layer
	GetLayer(workspaceName string, layerName string) (layer Layer, statusCode int)

	//UpdateLayer  update geoserver layer
	UpdateLayer(workspaceName string, layerName string, layer Layer) (modified bool, statusCode int)

	//DeleteLayer delete geoserver layer and its reources
	DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, statusCode int)
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
	Name         string   `json:"name,omitempty"`
	Path         string   `json:"path,omitempty"`
	Type         string   `json:"type,omitempty"`
	DefaultStyle Resource `json:"defaultStyle,omitempty"`
	Styles       struct {
		Class string     `json:"@class,omitempty"`
		Style []Resource `json:"style,omitempty"`
	}
	Resource    Resource    `json:"resource,omitempty"`
	Queryable   bool        `json:"queryable,omitempty"`
	Opaque      bool        `json:"opaque,omitempty"`
	Attribution Attribution `json:"attribution,omitempty"`
}

//LayerBody api json
type LayerBody struct {
	Layer Layer `json:"layer,omitempty"`
}

// GetshpFiledsName datastore name from shapefile name
func (g *GeoServer) GetshpFiledsName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name

}

// UploadShapeFile upload shapefile to geoserver
func (g *GeoServer) UploadShapeFile(fileURI string, WorkspaceName string, datastoreName string) ([]byte, int) {
	filename := filepath.Base(fileURI)
	if datastoreName == "" {
		datastoreName = g.GetshpFiledsName(filename)
	}
	targetURL := fmt.Sprintf("%srest/workspaces/%s/datastores/%s/file.shp",
		g.ServerURL,
		g.WorkspaceName,
		datastoreName)
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		g.logger.Fatal(err)
	}

	g.CreateWorkspace(WorkspaceName)
	return g.DoPut(targetURL, bytes.NewBuffer(shapeFileBinary), zipType, "")

}

//GetLayers  get all geoserver layers
func (g *GeoServer) GetLayers(workspaceName string) (layers []Resource, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	url := fmt.Sprintf("%srest/%slayers", g.ServerURL, workspaceName)
	response, responseCode := g.DoGet(url, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		layers = nil
		return
	}
	var layerResponse struct {
		Layers struct {
			Layer []Resource
		}
	}
	g.DeSerializeJSON(response, &layerResponse)
	layers = layerResponse.Layers.Layer
	return
}

//GetLayer  get geoserver layer
func (g *GeoServer) GetLayer(workspaceName string, layerName string) (layer Layer, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	url := fmt.Sprintf("%srest/%slayers/%s", g.ServerURL, workspaceName, layerName)
	response, responseCode := g.DoGet(url, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		layer = Layer{}
		return
	}
	var layerResponse struct {
		Layer Layer
	}
	g.DeSerializeJSON(response, &layerResponse)
	layer = layerResponse.Layer
	return
}

//UpdateLayer  update geoserver layer
func (g *GeoServer) UpdateLayer(workspaceName string, layerName string, layer Layer) (modified bool, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	url := fmt.Sprintf("%srest/%slayers/%s", g.ServerURL, workspaceName, layerName)
	data := LayerBody{Layer: layer}

	serializedLayer, _ := g.SerializeStruct(data)
	response, responseCode := g.DoPut(url, bytes.NewBuffer(serializedLayer), jsonType, jsonType)
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		modified = false
		return
	}
	modified = true
	return
}

//DeleteLayer delete geoserver layer and its reources
func (g *GeoServer) DeleteLayer(workspaceName string, layerName string, recurse bool) (deleted bool, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	url := fmt.Sprintf("%srest/%slayers/%s", g.ServerURL, workspaceName, layerName)
	response, responseCode := g.DoDelete(url, jsonType, map[string]string{"recurse": strconv.FormatBool(recurse)})
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		return
	}
	deleted = true
	return
}
