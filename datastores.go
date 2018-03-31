package geoserver

import (
	"bytes"
	"fmt"
	"strconv"
)

// DatastoreService define all geoserver datastore operations
type DatastoreService interface {
	// DatastoreExists checks if a datastore exists in a workspace
	DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error)

	// GetDatastores return datastores in a workspace
	GetDatastores(workspaceName string) (datastores []Resource, err error)

	// GetDatastoreDetails get specific datastore
	GetDatastoreDetails(workspaceName string, datastoreName string) (datastore Datastore, err error)

	//CreateDatastore create a datastore under provided workspace
	CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error)

	// DeleteDatastore deletes a datastore
	DeleteDatastore(workspaceName string, datastoreName string, recurse bool) (deleted bool, err error)
}

// Datastore holds geoserver store
type Datastore struct {
	Name                 string                    `json:",omitempty"`
	Href                 string                    `json:",omitempty"`
	Type                 string                    `json:",omitempty"`
	Enabled              bool                      `json:",omitempty"`
	Workspace            Workspace                 `json:",omitempty"`
	Default              bool                      `json:"_default,omitempty"`
	FeatureTypes         string                    `json:"featureTypes,omitempty"`
	ConnectionParameters DatastoreConnectionParams `json:"connectionParameters,omitempty"`
}

// DatastoreConnection holds paramters to create new datastore
type DatastoreConnection struct {
	Name   string
	Host   string
	Port   int
	DBName string
	DBUser string
	DBPass string
	Type   string
}

// ConnectionParamter is  item  in entry paramter in datastore connection paramters
type ConnectionParamter struct {
	Name  string `json:"@key"`
	Value string `json:"$"`
}

// DatastoreConnectionParams in datastore json
type DatastoreConnectionParams struct {
	Entry []ConnectionParamter `json:",omitempty"`
}

// ParseConnectionParameters convert from @key and $ to proper key and value
func (datastore *Datastore) ParseConnectionParameters() (paramters map[string]string) {
	paramters = make(map[string]string)
	if datastore.ConnectionParameters.Entry != nil {
		for _, paramter := range datastore.ConnectionParameters.Entry {
			paramters[paramter.Name] = paramter.Value
		}
	}
	return paramters

}

//DatastoreExists check if datastore in geoserver
func (g *GeoServer) DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error) {
	url := fmt.Sprintf("%s/rest/workspaces/%s/datastores/%s", g.ServerURL, workspaceName, datastoreName)
	_, responseCode := g.DoGet(url, jsonType, map[string]string{"quietOnNotFound": strconv.FormatBool(quietOnNotFound)})
	if responseCode != statusOk {
		exists = false
		err = statusErrorMapping[responseCode]
		return
	}
	exists = true
	return
}

//GetDatastores query geoserver datastores for current workspace
func (g *GeoServer) GetDatastores(workspaceName string) (datastores []Resource, err error) {
	//TODO: check if workspace exist before creating it
	var targetURL = fmt.Sprintf("%srest/workspaces/%s/datastores", g.ServerURL, workspaceName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		datastores = nil
		err = statusErrorMapping[responseCode]
		return
	}
	var query struct {
		DataStores struct {
			DataStore []Resource
		}
	}
	g.DeSerializeJSON(response, &query)
	datastores = query.DataStores.DataStore
	return
}

//GetDatastoreDetails query geoserver datastore for current workspace
func (g *GeoServer) GetDatastoreDetails(workspaceName string, datastoreName string) (datastore Datastore, err error) {
	//TODO: check if workspace exist before creating it
	var targetURL = fmt.Sprintf("%srest/workspaces/%s/datastores/%s", g.ServerURL, workspaceName, datastoreName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		datastore = Datastore{}
		err = statusErrorMapping[responseCode]
		return

	}
	type DatastoreDetails struct {
		Datastore Datastore `json:"dataStore"`
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
	targetURL := fmt.Sprintf("%s/rest/workspaces/%s/datastores", g.ServerURL, workspaceName)
	data := bytes.NewReader([]byte(xml))
	response, responseCode := g.DoPost(targetURL, data, xmlType, jsonType)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		err = statusErrorMapping[responseCode]
		return
	}
	created = true
	return

}

//DeleteDatastore delete geoserver datastore and its reources
func (g *GeoServer) DeleteDatastore(workspaceName string, datastoreName string, recurse bool) (deleted bool, err error) {
	url := fmt.Sprintf("%s/rest/workspaces/%s/datastores/%s", g.ServerURL, workspaceName, datastoreName)
	response, responseCode := g.DoDelete(url, jsonType, map[string]string{"recurse": strconv.FormatBool(recurse)})
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = statusErrorMapping[responseCode]
		return
	}
	deleted = true
	return
}
