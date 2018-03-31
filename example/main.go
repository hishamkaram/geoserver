package main

import (
	"fmt"

	"github.com/hishamkaram/geoserver"
)

var uploadedPath = "./uploaded/"

var gsCatalog geoserver.GeoServer

// //ContextData hold template data
// type ContextData struct {
// 	Geoserver geoserver.GeoServer
// 	Code      int
// }

// func handleUploaded(file *bytes.Buffer, filename string) string {
// 	_ = os.Mkdir(uploadedPath, 0700)
// 	filepath := uploadedPath + filename
// 	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer f.Close()
// 	io.Copy(f, file)
// 	return filepath
// }

// func index(w http.ResponseWriter, r *http.Request) {
// 	tmplData := ContextData{Geoserver: gsCatalog, Code: 0}
// 	if r.Method == "POST" {
// 		file, handler, err := r.FormFile("fileupload")
// 		defer file.Close()
// 		if err != nil {
// 			panic(err)
// 		}
// 		buf := bytes.NewBuffer(nil)
// 		if _, err := io.Copy(buf, file); err != nil {
// 			panic(err)
// 		}
// 		uploadedPath := handleUploaded(buf, handler.Filename)
// 		fileLocation, _ := filepath.Abs(uploadedPath)
// 		response, statusCode := gsCatalog.UploadShapeFile(fileLocation, "")
// 		tmplData.Code = statusCode
// 		fmt.Println(response, statusCode)
// 	}
// 	tmplt, _ := template.ParseFiles("templates/home.html")
// 	tmplt.Execute(w, tmplData)
// }
func main() {
	gsCatalog := geoserver.GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")

	layers, _ := gsCatalog.GetLayers("")
	for _, lyr := range layers {
		fmt.Printf("\nName:%s\t\nHref:%s\n", lyr.Name, lyr.Href)
	}
	//Test getLayer
	// layer, _ := gsCatalog.GetLayer("topp:tasmania_state_boundaries")
	// fmt.Printf("%s\t%s\t%s\t%s", layer.Name, strconv.FormatBool(layer.Opaque), layer.Path, layer.Type)
	//Test DeleteLayer
	// deleted, _ := gsCatalog.DeleteLayer("topp:tasmania_state_boundaries__2222", true)
	// fmt.Printf("\nDeleted:%s\n", strconv.FormatBool(deleted))

	//Test Updatelayer
	// modified, _ := gsCatalog.UpdateLayer("tiger:giant_polygon", geoserver.Layer{DefaultStyle: geoserver.Resource{Name: "giant_polygon", Href: "http://localhost:8080/geoserver13/rest/styles/giant_polygon.json"}})
	// fmt.Printf("\nDeleted:%s\n", strconv.FormatBool(modified))

	//Test if geoserver is running
	// isRunning, code := gsCatalog.IsRunning()
	// fmt.Printf("\nGeoserver status : %s \n StatusCode: %s\n", strconv.FormatBool(isRunning), strconv.Itoa(code))

	// Test get workspaces
	// workspaces, _ := gsCatalog.GetWorkspaces()
	// for _, workspace := range workspaces {
	// 	fmt.Printf("\nworkspace:\n name:%s\nhref:%s\n", workspace.Name, workspace.Href)
	// }

	//Test if workspace exist
	// exists, _ := gsCatalog.WorkspaceExists("NotFound")
	// fmt.Println(strconv.FormatBool(exists))

	//Test Create Style
	// created, _ := gsCatalog.CreateStyle("test_sld")
	// fmt.Println(strconv.FormatBool(created))

	//Test upload sld
	// sld, err := ioutil.ReadFile("sample/museum_nyc.sld") // just pass the file name
	// if err != nil {
	// 	fmt.Print(err)
	// }
	// style, _ := gsCatalog.UploadStyle(bytes.NewBuffer(sld), "test_sld")
	// fmt.Println(style)

	//Test Create Workspace
	// created, _ := gsCatalog.CreateWorkspace("test")
	// fmt.Println(strconv.FormatBool(created))

	//Test Create Workspace
	// created, _ := gsCatalog.CreateWorkspace("test")
	// fmt.Println(strconv.FormatBool(created))

	//Test Delete Workspace
	// deleted, _ := gsCatalog.DeleteWorkspace("test", true)
	// fmt.Println(strconv.FormatBool(deleted))

	//Test if datastore exist
	// exists, _ := gsCatalog.DatastoreExists("geonode", "cartoview_datastore", true)
	// fmt.Println(strconv.FormatBool(exists))

	// Test get datastores
	// datastores, _ := gsCatalog.GetDatastores("geonode")
	// for _, ds := range datastores {
	// 	fmt.Printf("\n datastores:\n name:%s\nhref:%s\n", ds.Name, ds.Href)
	// }

	// Test get specific datastore
	// datastore, _ := gsCatalog.GetDatastoreDetails("geonode", "cartoview_datastore")
	// fmt.Printf("\ndatastores:\nname:%s\nhref:%s\n", datastore.Name, datastore.Type)
	// for k, v := range datastore.ParseConnectionParameters() {
	// 	fmt.Printf("\nkey:%s \nvalue:%s", k, v)
	// }

	//Test Create datastore
	// ds := geoserver.DatastoreConnection{
	// 	Name:   "hisham",
	// 	Host:   "localhost",
	// 	Port:   5432,
	// 	DBName: "cartoview_datastore",
	// 	DBUser: "hishamkaram",
	// 	DBPass: "xxxx",
	// 	Type:   "postgis",
	// }
	// created, _ := gsCatalog.CreateDatastore(ds, "geonode")
	// fmt.Println(strconv.FormatBool(created))

	//Test Delete datastore
	// deleted, _ := gsCatalog.DeleteDatastore("geonode", "hisham", true)
	// fmt.Println(strconv.FormatBool(deleted))

	// r := mux.NewRouter()
	// r.HandleFunc("/", index)
	// s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	// r.PathPrefix("/static/").Handler(s)
	// http.Handle("/", r)
	// if err := http.ListenAndServe(":8081", nil); err != nil {
	// 	log.Fatal(err)
	// }
}
