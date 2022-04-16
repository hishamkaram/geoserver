package geoserver

import (
	"bytes"
	"net/http"
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

//DatastoreConnector interface to datastore connection object
type DatastoreConnector interface {
	GetDatastoreObj() (datastore Datastore)
}

// DatastoreConnection holds parameters to create new datastore in geoserver
type DatastoreConnection struct {
	Name           string
	Host           string
	Port           int
	DBName         string
	DBUser         string
	DBPass         string
	Type           string
	Schema         string
	MinConnections int
	MaxConnections int
}

// DatastoreConnectionParams in datastore json
type DatastoreConnectionParams struct {
	Entry []*Entry `json:"entry,omitempty"`
}

// DatastoreJNDIConnection holds parameters to create new datastore using JNDI connection pool
// see https://docs.geoserver.org/stable/en/user/tutorials/tomcat-jndi/tomcat-jndi.html
type DatastoreJNDIConnection struct {
	Name              string
	Type              string //dbtype
	JndiReferenceName string //
	Options           []Entry
}

//GetDatastoreObj return datastore Object to send to geoserver
func (connection DatastoreJNDIConnection) GetDatastoreObj() (datastore Datastore) {
	datastore = Datastore{
		Name: connection.Name,
		ConnectionParameters: DatastoreConnectionParams{
			Entry: []*Entry{
				{
					Key:   "jndiReferenceName",
					Value: connection.JndiReferenceName,
				},
				{
					Key:   "dbtype",
					Value: connection.Type,
				},
			},
		},
	}

	if connection.Options != nil {
		for i, _ := range connection.Options {
			datastore.ConnectionParameters.Entry = append(datastore.ConnectionParameters.Entry, &connection.Options[i])
		}
	}
	return
}

//GetDatastoreObj return datastore Object to send to geoserver rest
func (connection DatastoreConnection) GetDatastoreObj() (datastore Datastore) {

	dbSchema := connection.DBSchema
	if dbSchema == "" {
		dbSchema = "public"
	}

	datastore = Datastore{
		Name: connection.Name,
		ConnectionParameters: DatastoreConnectionParams{
			Entry: []*Entry{
				{
					Key:   "host",
					Value: connection.Host,
				},
				{
					Key:   "port",
					Value: strconv.Itoa(connection.Port),
				},
				{
					Key:   "database",
					Value: connection.DBName,
				},
				{
					Key:   "schema",
					Value: dbSchema,
				},
				{
					Key:   "user",
					Value: connection.DBUser,
				},
				{
					Key:   "passwd",
					Value: connection.DBPass,
				},
				{
					Key:   "dbtype",
					Value: connection.Type,
				},
			},
		},
	}
	if connection.Schema != "" {
		datastore.ConnectionParameters.Entry = append(datastore.ConnectionParameters.Entry,
			&Entry{
				Key:   "schema",
				Value: connection.Schema,
			},
		)
	}

	if connection.MinConnections != 0 {
		datastore.ConnectionParameters.Entry = append(datastore.ConnectionParameters.Entry,
			&Entry{
				Key:   "min connections",
				Value: strconv.Itoa(connection.MinConnections),
			},
		)
	}
	if connection.MaxConnections != 0 {
		datastore.ConnectionParameters.Entry = append(datastore.ConnectionParameters.Entry,
			&Entry{
				Key:   "max connections",
				Value: strconv.Itoa(connection.MaxConnections),
			},
		)
	}
	return
}

// DatastoreExists checks if a datastore exists in a workspace else return error
func (g *GeoServer) DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: http.MethodGet,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"quietOnNotFound": strconv.FormatBool(quietOnNotFound)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
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
		Method: http.MethodGet,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
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
		Method: http.MethodGet,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
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
func (g *GeoServer) CreateDatastore(datastoreConnection DatastoreConnector, workspaceName string) (created bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores")

	store := datastoreConnection.GetDatastoreObj()
	datastore := DatastoreDetails{
		Datastore: &store,
	}
	data, _ := g.SerializeStruct(datastore)
	httpRequest := HTTPRequest{
		Method:   http.MethodPost,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(data),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusCreated {
		//g.logger.Warn(string(response))
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
		Method: http.MethodDelete,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
		//g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
