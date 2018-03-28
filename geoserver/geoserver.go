package geoserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

//GeoServer is the configuration Object
type GeoServer struct {
	WorkspaceName string `yaml:"workspace"`
	ServerURL     string `yaml:"geoserver_url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
}

//Workspace is the Workspace Object
type Workspace struct {
	Name string
	Href string
}

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
func (g *GeoServer) doGet(url string, accept string) ([]byte, int) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	if accept != "" {
		req.Header.Add("Accept", fmt.Sprintf("%s", accept))
	}
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		panic(httpErr)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 201 {
			fmt.Printf("%s \n", string(body))
		}
		fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
		return body, resp.StatusCode
	}
}
func (g *GeoServer) doPost(url string, data []byte, dataType string) ([]byte, int) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	req.Header.Add("Content-Type", fmt.Sprintf("%s; charset=utf-8", dataType))
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		panic(httpErr)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 201 {
			fmt.Printf("%s \n", string(body))
		}
		fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
		return body, resp.StatusCode
	}

}
func (g *GeoServer) doPut(url string, data []byte, dataType string) ([]byte, int) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	req.Header.Add("Content-Type", fmt.Sprintf("%s", dataType))
	req.Header.Set("Accept", "application/xml")
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		panic(httpErr)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 201 {
			fmt.Printf("%s \n", string(body))
		}
		fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
		return body, resp.StatusCode
	}

}
func (g *GeoServer) createWorkspace() ([]byte, int) {
	//TODO: check if workspace exist before creating it
	var xml = fmt.Sprintf("<workspace><name>%s</name></workspace>", g.WorkspaceName)
	var targetURL = fmt.Sprintf("%srest/workspaces", g.ServerURL)
	data := []byte(xml)
	return g.doPost(targetURL, data, "text/xml")
}
func isEmpty(object interface{}) bool {
	if object == nil {
		return true
	} else if object == "" {
		return true
	} else if object == false {
		return true
	}
	if reflect.ValueOf(object).Kind() == reflect.Struct {
		empty := reflect.New(reflect.TypeOf(object)).Elem().Interface()
		if reflect.DeepEqual(object, empty) {
			return true
		}
	}
	return false
}

//GetDatastores query geoserver datastores for current workspace
func (g *GeoServer) GetDatastores() []Datastore {
	//TODO: check if workspace exist before creating it
	var targetURL = fmt.Sprintf("%srest/workspaces/%s/datastores", g.ServerURL, g.WorkspaceName)
	response, code := g.doGet(targetURL, "application/json")
	if code != 200 {
		log.Println(string(response))
	}
	var query DataStoreQuery
	err := json.Unmarshal([]byte(response), &query)
	if err != nil {
		panic(err)
	}
	if !isEmpty(query.DataStores) && (len(query.DataStores.DataStore) > 0) {
		return query.DataStores.DataStore
	}
	return nil
}

//GetDatastoreDetails query geoserver datastore for current workspace
func (g *GeoServer) GetDatastoreDetails(datastoreName string) Datastore {
	//TODO: check if workspace exist before creating it
	var targetURL = fmt.Sprintf("%srest/workspaces/%s/datastores/%s", g.ServerURL, g.WorkspaceName, datastoreName)
	response, code := g.doGet(targetURL, "application/json")
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
func (g *GeoServer) getShpdatastore(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name

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
	return g.doPost(targetURL, data, "text/xml")

}
func (g *GeoServer) shpFiledsName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name

}

//UploadShapeFile upload shapefile to geoserver
func (g *GeoServer) UploadShapeFile(fileURI string, datastoreName string) ([]byte, int) {
	filename := filepath.Base(fileURI)
	if datastoreName == "" {
		datastoreName = g.shpFiledsName(filename)
	}
	targetURL := fmt.Sprintf("%srest/workspaces/%s/datastores/%s/file.shp", g.ServerURL, g.WorkspaceName, datastoreName)
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		log.Fatal(err)
	}
	g.createWorkspace()
	return g.doPut(targetURL, shapeFileBinary, "application/zip")

}
