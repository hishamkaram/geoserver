package geoserver

import "net/http"

// Catalog is geoserver interface that define all operatoins
type Catalog interface {
	WorkspaceService
	DatastoreService
	StyleService
	AboutService
	LayerService
	LayerGroupService
	CoverageStoresService
	FeatureTypeService
	UtilsInterface
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
		HttpClient: &http.Client{},
		logger:     GetLogger(),
	}
	return &geoserver
}
