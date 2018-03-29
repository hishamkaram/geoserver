package geoserver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Catalog is geoserver interface that define all operatoins
type Catalog interface {
	WorkspaceExists(workspaceName string) (exists bool, statusCode int)

	// GetWorkspaces get geoserver workspaces
	GetWorkspaces() (workspaces []Workspace, statusCode int)

	// CreateWorkspace creates a workspace
	CreateWorkspace(workspaceName string) (created bool, statusCode int)

	// DeleteWorkspace deletes a workspace
	DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, statusCode int)

	// DatastoreExists checks if a datastore exists in a workspace
	DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, statusCode int)
	// GetDatastores return datastores in a workspace
	GetDatastores(workspaceName string) (datastores []Datastore, statusCode int)
	// GetDatastoreDetails get specific datastore
	GetDatastoreDetails(workspaceName string, datastoreName string) (datastore Datastore, statusCode int)

	//CreateDatastore create a datastore under provided workspace
	CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, statusCode int)

	// TODO:implement
	DeleteDatastore(Datastore) error

	//IsRunning check if geoserver is running return true and statusCode of request
	IsRunning() (running bool, statusCode int)

	// GetshpFiledsName datastore name from shapefile name
	GetShpdatastore(filename string) string

	// UploadShapeFile upload shapefile to geoserver
	UploadShapeFile(fileURI string, WorkspaceName string, datastoreName string) ([]byte, int)

	// TODO:implement
	// FeatureTypeExists

	// TODO:implement
	// GetFeatureTypes

	// TODO:implement
	// CreateFeatureType

	// TODO:implement
	// DeleteFeatureType

	// TODO:implement
	// UploadSLD

	// TODO:implement
	// ChangeSLD

	// TODO:implement
	// DeleteSLD
}

//GeoServer is the configuration Object
type GeoServer struct {
	WorkspaceName string `yaml:"workspace"`
	ServerURL     string `yaml:"geoserver_url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HTTPClient    *http.Client
}

//LoadConfig load geoserver config from yaml file
func (g *GeoServer) LoadConfig(configFile string) *GeoServer {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, g)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return g
}

// GetGeoserverRequest creates a HTTP request with geoserver credintails and header
func (g *GeoServer) GetGeoserverRequest(targetURL string, method string, accept string, data io.Reader, contentType string) (request *http.Request, err error) {
	request, err = http.NewRequest(method, targetURL, data)
	if err != nil {
		return
	}
	if data != nil {
		request.Header.Set(contentTypeHeader, contentType)
	}
	if accept != "" {
		request.Header.Set(acceptHeader, accept)
	}

	request.SetBasicAuth(g.Username, g.Password)
	return request, err
}

// IsRunning check if geoserver is running
func (g *GeoServer) IsRunning() (running bool, statusCode int) {
	url := fmt.Sprintf("%srest/about/version", g.ServerURL)
	_, responseCode := g.DoGet(url, jsonType, nil)
	if responseCode != statusOk {
		running = false
		statusCode = responseCode
		return
	}
	running = true
	statusCode = responseCode
	return
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
	targetURL := fmt.Sprintf("%srest/workspaces/%s/datastores/%s/file.shp", g.ServerURL, g.WorkspaceName, datastoreName)
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		log.Fatal(err)
	}

	g.CreateWorkspace(WorkspaceName)
	return g.DoPut(targetURL, bytes.NewBuffer(shapeFileBinary), zipType, "")

}
