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

//GetCatalog return geoserver catalog instance
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
