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
			fmt.Println("Successfully Uploaded")
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
		}

	}
}
