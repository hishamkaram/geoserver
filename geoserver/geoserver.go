package geoserver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
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
func (g *GeoServer) doPost(url string, data []byte, dataType string) (bool, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	req.Header.Add("Content-Type", fmt.Sprintf("%s; charset=utf-8", dataType))
	resp, httpErr := client.Do(req)
	if err != nil {
		log.Fatal(httpErr)
		return false, httpErr
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 201 {

		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("%s \n", string(body))
		}
	}
	fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
	return true, nil

}
func (g *GeoServer) doPut(url string, data []byte, dataType string) (bool, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	req.Header.Add("Content-Type", fmt.Sprintf("%s", dataType))
	req.Header.Set("Accept", "application/xml")
	resp, httpErr := client.Do(req)
	if err != nil {
		log.Fatal(httpErr)
		return false, httpErr
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != 201 {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("%s \n", string(body))
		}
	}
	fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
	return true, nil

}
func (g *GeoServer) createWorkspace() (bool, error) {
	//TODO: check if workspace exist before creating it
	var xml = fmt.Sprintf("<workspace><name>%s</name></workspace>", g.WorkspaceName)
	var targetURL = fmt.Sprintf("%srest/workspaces", g.ServerURL)
	data := []byte(xml)
	success, err := g.doPost(targetURL, data, "text/xml")
	if err != nil {
		log.Fatal(err)
	}
	return success, err
}
func (g *GeoServer) getShpdatastore(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name

}

//CreateDataStore create a datastore under current workspace
func (g *GeoServer) CreateDataStore(name string, dbName string, host string, port string, dbUser string, dbPass string) (bool, error) {
	//TODO: check if data exist before creating it
	rawXML := `<dataStore>
				<name>%s</name>
				<connectionParameters>
				<host>%s</host>
				<port>%d</port>
				<database>%s</database>
				<user>%s</user>
				<passwd>%s</passwd>
				<dbtype>%s</dbtype>
				</connectionParameters>
			</dataStore>`
	xml := fmt.Sprintf(rawXML, name, host, port, dbName, dbUser, dbPass, "postgis")
	targetURL := fmt.Sprintf("%s/rest/workspaces/%s/datastores", g.ServerURL, g.WorkspaceName)
	data := []byte(xml)
	success, err := g.doPost(targetURL, data, "text/xml")
	if err != nil {
		log.Fatal(err)
	}
	return success, err

}
func (g *GeoServer) shpFiledsName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name

}

//UploadShapeFile upload shapefile to geoserver
func (g *GeoServer) UploadShapeFile(fileURI string, dataStoreName string) (bool, error) {
	filename := filepath.Base(fileURI)
	if dataStoreName == "" {
		dataStoreName = g.shpFiledsName(filename)
	}
	targetURL := fmt.Sprintf("%srest/workspaces/%s/datastores/%s/file.shp", g.ServerURL, g.WorkspaceName, dataStoreName)
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		log.Fatal(err)
	}
	_, workspaceErr := g.createWorkspace()
	if workspaceErr != nil {
		log.Fatal(err)
	}
	success, uploadErr := g.doPut(targetURL, shapeFileBinary, "application/zip")
	return success, uploadErr

}
