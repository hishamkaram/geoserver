package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

var geoserverURL = "http://localhost:8080/geoserver/"
var workspace = "golang"
var uploadedPath = "./uploaded/"

//GeoServer is configuration struct
type GeoServer struct {
	WorkspaceName string
	ServerUrl     string
	Username      string
	Password      string
}

var geoserver = GeoServer{WorkspaceName: workspace, ServerUrl: geoserverURL, Username: "admin", Password: "geoserver"}

func handleUploaded(file *bytes.Buffer, filename string) string {
	_ = os.Mkdir(uploadedPath, 0700)
	filepath := uploadedPath + filename
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(f, file)
	return filepath
}
func index(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		file, handler, err := r.FormFile("fileupload")
		defer file.Close()
		if err != nil {
			log.Fatal(err)
		}
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			panic(err)
		}
		uploadedPath := handleUploaded(buf, handler.Filename)
		uploadShapeFile(uploadedPath)
	}
	tmplt := template.New("home.html")
	tmplt, _ = tmplt.ParseFiles("templates/home.html")

	tmplt.Execute(w, geoserver)
}
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", index)
	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	r.PathPrefix("/static/").Handler(s)
	http.Handle("/", r)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
func createWorkspace(workspaceName string) (bool, error) {
	var xml = fmt.Sprintf("<workspace><name>%s</name></workspace>", workspaceName)
	var targetURL = fmt.Sprintf("%srest/workspaces", geoserver.ServerUrl)
	client := &http.Client{}
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer([]byte(xml)))
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(geoserver.Username, geoserver.Password)
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
	targetURL := fmt.Sprintf("%srest/workspaces/%s/datastores/%s/file.shp", geoserver.ServerUrl, geoserver.WorkspaceName, datastoreName(filename))
	shapeFileBinary, err := ioutil.ReadFile(fileURI)
	if err != nil {
		log.Fatal(err)
	}
	_, workspaceErr := createWorkspace(geoserver.WorkspaceName)
	if workspaceErr != nil {
		log.Fatal(err)
	}
	req, err := http.NewRequest("PUT", targetURL, bytes.NewBuffer(shapeFileBinary))
	req.SetBasicAuth(geoserver.Username, geoserver.Password)
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("Accept", "application/xml")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	} else {
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		responseCode, err := strconv.ParseInt(strings.TrimSpace(resp.Status), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		if responseCode == 201 {
			fmt.Println("Layer Uploaded Successfully")
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
		}

	}

}
