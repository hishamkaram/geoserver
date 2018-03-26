package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var geoserverURL = "http://localhost:8080/geoserver/"
var workspace = "golang"

func main() {
	fmt.Print("Enter Zip File Path: ")
	var filePATH string
	fmt.Scanln(&filePATH)
	uploadShapeFile(filePATH)
}
func createWorkspace(workspaceName string) (bool, error) {
	var xml = fmt.Sprintf("<workspace><name>%s</name></workspace>", workspaceName)
	var targetURL = fmt.Sprintf("%srest/workspaces", geoserverURL)
	client := &http.Client{}
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer([]byte(xml)))
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth("admin", "geoserver")
	req.Header.Add("Content-Type", "text/xml; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false, err
	} else {
		defer resp.Body.Close()
		responseCode, err := strconv.ParseInt(strings.TrimSpace(resp.Status), 10, 64)
		if err != nil {
			return false, err
		}
		if responseCode == 201 {
			fmt.Printf("workspace: %s Created Successfully \n", workspaceName)
			return true, nil

		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
			return false, err
		}
	}
}
func datastoreName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name

}
func uploadShapeFile(fileURI string) {
	filename := filepath.Base(fileURI)
	targetURL := fmt.Sprintf("%srest/workspaces/%s/datastores/%s/file.shp", geoserverURL, workspace, datastoreName(filename))
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		panic(err)
	}
	_, workspaceErr := createWorkspace(workspace)
	if workspaceErr != nil {
		panic(err)
	}
	req, err := http.NewRequest("PUT", targetURL, bytes.NewBuffer(shapeFileBinary))
	req.SetBasicAuth("admin", "geoserver")
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("Accept", "application/xml")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	} else {
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		responseCode, err := strconv.ParseInt(strings.TrimSpace(resp.Status), 10, 64)
		if err != nil {
			panic(err)
		}
		if responseCode == 201 {
			fmt.Println("Layer Uploaded Successfully")
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
		}

	}

}
