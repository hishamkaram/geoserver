package geoserver

import (
	"bytes"
	"fmt"
	"strconv"
)

// CoverageStoresService define all geoserver CoverageStores operations
type CoverageStoresService interface {

	// GetCoverageStores return all coverage store as resources
	GetCoverageStores(workspaceName string) (coverageStores []Resource, err error)

	// CreateCoverageStore create coverage store in geoserver and return created one else return error
	CreateCoverageStore(workspaceName string, coverageStore CoverageStore) (newCoverageStore CoverageStore, err error)

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
	Enabled     string    `json:"enabled,omitempty"`
	Workspace   Workspace `json:"workspace,omitempty"`
	Default     bool      `json:"_default,omitempty"`
	Coverages   string    `json:"coverages,omitempty"`
}

//CoverageStoreRequestBody geoserver coverage store to send to api
type CoverageStoreRequestBody struct {
	CoverageStore CoverageStore `json:"coverageStore,omitempty"`
}

// GetCoverageStores return all coverage store as resources
func (g *GeoServer) GetCoverageStores(workspaceName string) (coverageStores []Resource, err error) {
	targetURL := fmt.Sprintf("%srest/workspaces/%s/coveragestores", g.ServerURL, workspaceName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		coverageStores = nil
		err = statusErrorMapping[responseCode]
		return
	}
	var coverageStoresResponse struct {
		CoverageStores struct {
			CoverageStore []Resource
		}
	}
	g.DeSerializeJSON(response, &coverageStoresResponse)
	coverageStores = coverageStoresResponse.CoverageStores.CoverageStore
	return
}

// CreateCoverageStore create coverage store in geoserver and return created one else return error
func (g *GeoServer) CreateCoverageStore(workspaceName string, coverageStore CoverageStore) (newCoverageStore CoverageStore, err error) {
	targetURL := fmt.Sprintf("%srest/workspaces/%s/coveragestores", g.ServerURL, workspaceName)
	data := CoverageStoreRequestBody{
		CoverageStore: coverageStore,
	}
	serializedData, _ := g.SerializeStruct(data)
	response, responseCode := g.DoPost(targetURL, bytes.NewBuffer(serializedData), jsonType+"; charset=utf-8", jsonType)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		newCoverageStore = CoverageStore{}
		err = statusErrorMapping[responseCode]
		return
	}
	g.DeSerializeJSON(response, newCoverageStore)
	return
}

// UpdateCoverageStore  parital update coverage store in geoserver else return error
func (g *GeoServer) UpdateCoverageStore(workspaceName string, coverageStore CoverageStore) (modified bool, err error) {
	targetURL := fmt.Sprintf("%srest/workspaces/%s/coveragestores/%s", g.ServerURL, workspaceName, coverageStore.Name)
	data := CoverageStoreRequestBody{CoverageStore: coverageStore}
	serializedData, _ := g.SerializeStruct(data)
	response, responseCode := g.DoPut(targetURL, bytes.NewBuffer(serializedData), jsonType, jsonType)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		modified = false
		err = statusErrorMapping[responseCode]
		return
	}
	modified = true
	return
}

// DeleteCoverageStore delete coverage store from geoserver else return error
func (g *GeoServer) DeleteCoverageStore(workspaceName string, coverageStore string, recurse bool) (deleted bool, err error) {
	targetURL := fmt.Sprintf("%srest/workspaces/%s/coveragestores/%s", g.ServerURL, workspaceName, coverageStore)
	response, responseCode := g.DoDelete(targetURL, jsonType, map[string]string{"recurse": strconv.FormatBool(recurse)})
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = statusErrorMapping[responseCode]
		return
	}
	deleted = true
	return
}
