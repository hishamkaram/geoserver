package geoserver

import (
	"bytes"
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
	Name                 string                    `json:"name,omitempty"`
	Href                 string                    `json:"href,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Enabled              bool                      `json:"enabled,omitempty"`
	Workspace            *Workspace                `json:"workspace,omitempty"`
	Default              bool                      `json:"_default,omitempty"`
	FeatureTypes         string                    `json:"featureTypes,omitempty"`
	ConnectionParameters DatastoreConnectionParams `json:"connectionParameters,omitempty"`
}

//DatastoreDetails this struct to send and accept json data from/to geoserver
type DatastoreDetails struct {
	Datastore *Datastore `json:"dataStore"`
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
	Entry []*Entry `json:"entry,omitempty"`
}

//GetDatastoreObj return datastore Object to send to geoserver rest
func (connection *DatastoreConnection) GetDatastoreObj() (datastore Datastore) {
	datastore = Datastore{
		Name: connection.Name,
		ConnectionParameters: DatastoreConnectionParams{
			Entry: []*Entry{
				&Entry{
					Key:   "host",
					Value: connection.Host,
				},
				&Entry{
					Key:   "port",
					Value: strconv.Itoa(connection.Port),
				},
				&Entry{
					Key:   "database",
					Value: connection.DBName,
				},
				&Entry{
					Key:   "user",
					Value: connection.DBUser,
				},
				&Entry{
					Key:   "passwd",
					Value: connection.DBPass,
				},
				&Entry{
					Key:   "dbtype",
					Value: connection.Type,
				},
			},
		},
	}
	return
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
	var query DatastoreDetails
	g.DeSerializeJSON(response, &query)
	datastore = query.Datastore
	return

}

//CreateDatastore create a datastore under provided workspace
func (g *GeoServer) CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores")
	store := datastoreConnection.GetDatastoreObj()
	datastore := DatastoreDetails{
		Datastore: &store,
	}
	data, _ := g.SerializeStruct(datastore)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(data),
		DataType: jsonType,
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
