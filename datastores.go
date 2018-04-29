package geoserver

import (
	"bytes"
	"fmt"
	"strconv"
)

// DatastoreService define all geoserver datastore operations
type DatastoreService interface {

	// DatastoreExists checks if a datastore exists in a workspace else return error
	DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error)

	// GetDatastores return datastores in a workspace else return error
	GetDatastores(workspaceName string) (datastores []*Resource, err error)

	// GetDatastoreDetails get specific datastore from geoserver else return error
	GetDatastoreDetails(workspaceName string, datastoreName string) (datastore *Datastore, err error)

	//CreateDatastore create a datastore under provided workspace
	CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error)

	// DeleteDatastore deletes a datastore from geoserver else return error
	DeleteDatastore(workspaceName string, datastoreName string, recurse bool) (deleted bool, err error)
}

// Datastore holds geoserver store information
type Datastore struct {
	Name                 string     `json:"name,omitempty"`
	Href                 string     `json:"href,omitempty"`
	Type                 string     `json:"type,omitempty"`
	Enabled              bool       `json:"enabled,omitempty"`
	Workspace            *Workspace `json:"workspace,omitempty"`
	Default              bool       `json:"_default,omitempty"`
	FeatureTypes         string     `json:"featureTypes,omitempty"`
	ConnectionParameters *Entry     `json:"connectionParameters,omitempty"`
}

// DatastoreConnection holds parameters to create new datastore in geoserver
type DatastoreConnection struct {
	Name   string
	Host   string
	Port   int
	DBName string
	DBUser string
	DBPass string
	Type   string
}

// DatastoreConnectionParams in datastore json
type DatastoreConnectionParams struct {
	Entry []*Entry `json:",omitempty"`
}

// DatastoreExists checks if a datastore exists in a workspace else return error
func (g *GeoServer) DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"quietOnNotFound": strconv.FormatBool(quietOnNotFound)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		exists = false
		err = g.GetError(responseCode, response)
		return
	}
	exists = true
	return
}

// GetDatastores return datastores in a workspace else return error
func (g *GeoServer) GetDatastores(workspaceName string) (datastores []*Resource, err error) {
	//TODO: check if workspace exist before creating it
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		datastores = nil
		err = g.GetError(responseCode, response)
		return
	}
	var query struct {
		DataStores struct {
			DataStore []*Resource
		}
	}
	g.DeSerializeJSON(response, &query)
	datastores = query.DataStores.DataStore
	return
}

// GetDatastoreDetails get specific datastore from geoserver else return error
func (g *GeoServer) GetDatastoreDetails(workspaceName string, datastoreName string) (datastore *Datastore, err error) {
	//TODO: check if workspace exist before creating it
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		datastore = &Datastore{}
		err = g.GetError(responseCode, response)
		return

	}
	type DatastoreDetails struct {
		Datastore *Datastore `json:"dataStore"`
	}
	var query DatastoreDetails
	g.DeSerializeJSON(response, &query)
	datastore = query.Datastore
	return

}

//CreateDatastore create a datastore under provided workspace
func (g *GeoServer) CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error) {
	//TODO: check if data exist before creating it
	rawXML := `<dataStore>
				<name>%s</name>
				<connectionParameters>
				<host>%s</host>
				<port>%s</port>
				<database>%s</database>
				<user>%s</user>
				<passwd>%s</passwd>
				<dbtype>%s</dbtype>
				</connectionParameters>
			</dataStore>`
	xml := fmt.Sprintf(rawXML,
		datastoreConnection.Name,
		datastoreConnection.Host,
		strconv.Itoa(datastoreConnection.Port),
		datastoreConnection.DBName,
		datastoreConnection.DBUser,
		datastoreConnection.DBPass,
		datastoreConnection.Type)
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores")
	data := bytes.NewReader([]byte(xml))
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     data,
		DataType: xmlType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return

}

// DeleteDatastore deletes a datastore from geoserver else return error
func (g *GeoServer) DeleteDatastore(workspaceName string, datastoreName string, recurse bool) (deleted bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
