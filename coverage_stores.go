package geoserver

import (
	"bytes"
	"strconv"
)

// CoverageStoresService define all geoserver CoverageStores operations
type CoverageStoresService interface {

	// GetCoverageStores return all coverage store as resources
	GetCoverageStores(workspaceName string) (coverageStores []*Resource, err error)

	// CreateCoverageStore create coverage store in geoserver and return created one else return error
	CreateCoverageStore(workspaceName string, coverageStore CoverageStore) (created bool, err error)

	// UpdateCoverageStore  parital update coverage store in geoserver else return error
	UpdateCoverageStore(workspaceName string, coverageStore CoverageStore) (modified bool, err error)

	// DeleteCoverageStore delete coverage store from geoserver else return error
	DeleteCoverageStore(workspaceName string, coverageStore string, recurse bool) (deleted bool, err error)
}

//CoverageStore geoserver coverage store
type CoverageStore struct {
	Name        string    `json:"name,omitempty"`
	URL         string    `json:"url,omitempty"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type,omitempty"`
	Enabled     bool      `json:"enabled,omitempty"`
	Workspace   *Resource `json:"workspace,omitempty"`
	Default     bool      `json:"_default,omitempty"`
	Coverages   string    `json:"coverages,omitempty"`
}

//CoverageStoreRequestBody geoserver coverage store to send to api
type CoverageStoreRequestBody struct {
	CoverageStore *CoverageStore `json:"coverageStore,omitempty"`
}

// GetCoverageStores return all coverage store as resources,
// err is an error if error occurred else err is nil
func (g *GeoServer) GetCoverageStores(workspaceName string) (coverageStores []*Resource, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		coverageStores = nil
		err = g.GetError(responseCode, response)
		return
	}
	var coverageStoresResponse struct {
		CoverageStores struct {
			CoverageStore []*Resource
		}
	}
	g.DeSerializeJSON(response, &coverageStoresResponse)
	coverageStores = coverageStoresResponse.CoverageStores.CoverageStore
	return
}

// CreateCoverageStore create coverage store in geoserver and return created one else return error
func (g *GeoServer) CreateCoverageStore(workspaceName string, coverageStore CoverageStore) (created bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores")
	data := CoverageStoreRequestBody{
		CoverageStore: &coverageStore,
	}
	serializedData, _ := g.SerializeStruct(data)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Data:     bytes.NewBuffer(serializedData),
		DataType: jsonType + "; charset=utf-8",
		Accept:   jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// UpdateCoverageStore  parital update coverage store in geoserver else return error
func (g *GeoServer) UpdateCoverageStore(workspaceName string, coverageStore CoverageStore) (modified bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStore.Name)
	data := CoverageStoreRequestBody{CoverageStore: &coverageStore}
	serializedData, _ := g.SerializeStruct(data)
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Data:     bytes.NewBuffer(serializedData),
		DataType: jsonType,
		Accept:   jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		modified = false
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

// DeleteCoverageStore delete coverage store from geoserver else return error
func (g *GeoServer) DeleteCoverageStore(workspaceName string, coverageStore string, recurse bool) (deleted bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStore)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
