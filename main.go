package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/hishamkaram/gsbinding/geoserver"
)

var uploadedPath = "./uploaded/"

var gsCatalog geoserver.GeoServer

//ContextData hold template data
type ContextData struct {
	Geoserver geoserver.GeoServer
	Code      int
}

func handleUploaded(file *bytes.Buffer, filename string) string {
	_ = os.Mkdir(uploadedPath, 0700)
	filepath := uploadedPath + filename
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	io.Copy(f, file)
	return filepath
}
func index(w http.ResponseWriter, r *http.Request) {
	tmplData := ContextData{Geoserver: gsCatalog, Code: 0}
	if r.Method == "POST" {
		file, handler, err := r.FormFile("fileupload")
		defer file.Close()
		if err != nil {
			panic(err)
		}
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			panic(err)
		}
		uploadedPath := handleUploaded(buf, handler.Filename)
		fileLocation, _ := filepath.Abs(uploadedPath)
		response, statusCode := gsCatalog.UploadShapeFile(fileLocation, "")
		tmplData.Code = statusCode
		fmt.Println(response, statusCode)
	}
	tmplt, _ := template.ParseFiles("templates/home.html")
	tmplt.Execute(w, tmplData)
}
func main() {
	fileLocation, _ := filepath.Abs("./config.yml")
	gsCatalog.LoadConfig(fileLocation)
	r := mux.NewRouter()
	r.HandleFunc("/", index)
	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	r.PathPrefix("/static/").Handler(s)
	http.Handle("/", r)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
