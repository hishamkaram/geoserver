package geoserver

import "time"

// defaultHTTPTimeout is applied to the http.Client constructed by GetCatalog
// when no override is supplied. It is intentionally generous so that large
// shapefile/raster uploads do not hit a request-level timeout; per-call
// timeouts should be expressed via context (see *Context method variants).
const defaultHTTPTimeout = 30 * time.Second

// Catalog is the geoserver interface that bundles all REST resource services.
//
// Embedding new services here is additive in v1.x and only affects third-party
// fakes that implement the entire Catalog interface (rare). New consumers
// should hold a *GeoServer and call methods directly, or — for context-aware
// usage — the parallel CatalogWithContext interface (v1.1+).
type Catalog interface {
	WorkspaceService
	DatastoreService
	StyleService
	AboutService
	LayerService
	LayerGroupService
	CoverageStoresService
	CoverageService
	FeatureTypeService
	SettingsService
	SecurityService
	ACLService
	UtilsInterface
}

// GetCatalog returns a [GeoServer] catalog instance configured with the given
// base URL and basic-auth credentials.
//
// The returned client uses a [*http.Client] with a 30-second default timeout.
// Per-request cancellation should be expressed through the *Context method
// variants introduced in v1.1.
//
// Deprecated: prefer [New], which takes functional options for HTTP client,
// timeout, logger, etc. GetCatalog will be removed in v2.
func GetCatalog(geoserverURL string, username string, password string) (catalog *GeoServer) {
	return New(geoserverURL, username, password)
}
