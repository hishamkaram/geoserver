package geoserver

// Catalog is geoserver interface that define all operatoins
type Catalog interface {
	WorkspaceService
	DatastoreService
	FeatureTypeService
	StyleService
	AboutService
	LayerService
}
