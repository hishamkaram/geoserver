package main

import (
	"github.com/hishamkaram/geoserver"
)

var uploadedPath = "./uploaded/"

var gsCatalog geoserver.GeoServer

func main() {
	// gsCatalog := geoserver.GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	//Test getLayer
	// layers, err := gsCatalog.GetLayers("")
	// if err != nil {
	// 	fmt.Printf("\nError:%s\n", err)
	// }
	// for _, lyr := range layers {
	// 	fmt.Printf("\nName:%s  href:%s\n", lyr.Name, lyr.Href)
	// }

	//Test getLayer
	// layer, err := gsCatalog.GetLayer("nurc", "Arc_Sample")
	// if err != nil {
	// 	fmt.Printf("\nError:%s\n", err)
	// } else {
	// 	fmt.Printf("%+v\n", layer)
	// }

	// Test DeleteLayer
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

	// Test Create Style
	// created, err := gsCatalog.CreateStyle("geonode", "museum_nyc")
	// if err != nil {
	// 	fmt.Printf("\nError:%s\n", err)
	// }
	// fmt.Println(strconv.FormatBool(created))

	//Test upload sld
	// data, err := ioutil.ReadFile("sample/museum_nyc.sld")
	// if err != nil {
	// 	fmt.Print(err)
	// }
	// fmt.Println(string(data))
	// success, sldErr := gsCatalog.UploadStyle(bytes.NewBuffer(data), "geonode", "museum_nyc")
	// if sldErr != nil {
	// 	fmt.Print(err)
	// }
	// fmt.Println(strconv.FormatBool(success))

	// Test Create Workspace
	// created, err := gsCatalog.CreateWorkspace("golang")
	// if err != nil {
	// 	fmt.Printf("\nError:%s\n", err)
	// }
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
