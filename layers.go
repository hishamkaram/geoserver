package geoserver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

// LayerService define  geoserver layers operations
type LayerService interface {
	// GetshpFiledsName datastore name from shapefile name
	GetShpdatastore(filename string) string

	// UploadShapeFile upload shapefile to geoserver
	UploadShapeFile(fileURI string, WorkspaceName string, datastoreName string) ([]byte, int)
}

//Resource geoserver resource
type Resource struct {
	Class string `json:"@class,omitempty"`
	Name  string
	Href  string
}

//Layer geoserver layers
type Layer struct {
	Name         string `json:",omitempty"`
	Path         string `json:",omitempty"`
	Type         string `json:",omitempty"`
	DefaultStyle Style
	Styles       struct {
		Class string `json:"@class,omitempty"`
		Style []Style
	}
	Resource Resource
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
		log.Fatal(err)
	}

	g.CreateWorkspace(WorkspaceName)
	return g.DoPut(targetURL, bytes.NewBuffer(shapeFileBinary), zipType, "")

}
