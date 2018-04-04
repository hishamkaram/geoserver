package geoserver

import "net/http"

// Catalog is geoserver interface that define all operatoins
type Catalog interface {
	WorkspaceService
	DatastoreService
	StyleService
	AboutService
	LayerService
	CoverageStoresService
}

//GetCatalog return geoserver catalog instance,
//this fuction take geoserverURL('http://localhost:8080/geoserver/') ,
//geoserver username,
//geoserver password
// return geoserver structObj
func GetCatalog(geoserverURL string, username string, password string) (catalog *GeoServer) {
	geoserver := GeoServer{
		ServerURL:  geoserverURL,
		Username:   username,
		Password:   password,
		httpClient: &http.Client{},
	}
	geoserver.SetLogger()
	return &geoserver
}
