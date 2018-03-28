package geoserver

import (
	"encoding/json"
	"fmt"
	"log"
)

// Datastore holds geoserver store
type Datastore struct {
	Name         string
	Href         string
	Type         string
	Enabled      bool
	workspace    Workspace
	Default      bool `json:"_default"`
	featureTypes string
}

// Datastores holds a list of geoserver stores
type Datastores struct {
	DataStore []Datastore
}

// DataStoreQuery holds datastores query ("api json")
type DataStoreQuery struct {
	DataStores Datastores
}

//GetDatastores query geoserver datastores for current workspace
func (g *GeoServer) GetDatastores() []Datastore {
	//TODO: check if workspace exist before creating it
	var targetURL = fmt.Sprintf("%srest/workspaces/%s/datastores", g.ServerURL, g.WorkspaceName)
	response, code := g.DoGet(targetURL, "application/json")
	if code != 200 {
		log.Println(string(response))
	}
	var query DataStoreQuery
	err := json.Unmarshal([]byte(response), &query)
	if err != nil {
		panic(err)
	}
	if !IsEmpty(query.DataStores) && (len(query.DataStores.DataStore) > 0) {
		return query.DataStores.DataStore
	}
	return nil
}

//GetDatastoreDetails query geoserver datastore for current workspace
func (g *GeoServer) GetDatastoreDetails(datastoreName string) Datastore {
	//TODO: check if workspace exist before creating it
	var targetURL = fmt.Sprintf("%srest/workspaces/%s/datastores/%s", g.ServerURL, g.WorkspaceName, datastoreName)
	response, code := g.DoGet(targetURL, "application/json")
	if code != 200 {
		log.Println(string(response))
	}
	type DatastoreDetails struct {
		Datastore Datastore `json:"dataStore"`
	}
	var query DatastoreDetails
	err := json.Unmarshal([]byte(response), &query)
	if err != nil {
		panic(err)
	}
	return query.Datastore
}

//CreateDatastore create a datastore under current workspace
func (g *GeoServer) CreateDatastore(name string, dbName string, host string, port string, dbUser string, dbPass string) ([]byte, int) {
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
	xml := fmt.Sprintf(rawXML, name, host, port, dbName, dbUser, dbPass, "postgis")
	targetURL := fmt.Sprintf("%s/rest/workspaces/%s/datastores", g.ServerURL, g.WorkspaceName)
	data := []byte(xml)
	return g.DoPost(targetURL, data, "text/xml")

}
