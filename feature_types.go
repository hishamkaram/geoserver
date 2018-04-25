package geoserver

import "fmt"

// FeatureTypeService define all geoserver featuretype operations
type FeatureTypeService interface {
	// TODO:implement
	// FeatureTypeExists

	GetFeatureTypes(workspaceName string, datastoreName string) (featureTypes []*Resource, err error)

	// TODO:implement
	// CreateFeatureType

	// TODO:implement
	// DeleteFeatureType
}

// FeatureTypes holds a list of geoserver styles
type FeatureTypes struct {
	FeatureType []*Resource `json:"featureType,omitempty"`
}

//FeatureTypesResponseBody is the api body
type FeatureTypesResponseBody struct {
	FeatureTypes *FeatureTypes `json:"featureTypes,omitempty"`
}

// GetFeatureTypes return all featureTypes in workspace and datastore if error occured err will be return and nil for featrueTypes
func (g *GeoServer) GetFeatureTypes(workspaceName string, datastoreName string) (featureTypes []*Resource, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	if datastoreName != "" {
		datastoreName = fmt.Sprintf("datastores/%s/featuretypes", datastoreName)
	}
	targetURL := g.ParseURL("rest", workspaceName, datastoreName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		featureTypes = nil
		err = statusErrorMapping[responseCode]
		return
	}
	featureTypesResponse := &FeatureTypesResponseBody{FeatureTypes: &FeatureTypes{FeatureType: make([]*Resource, 0, 0)}}
	g.DeSerializeJSON(response, featureTypesResponse)
	featureTypes = featureTypesResponse.FeatureTypes.FeatureType
	return
}
