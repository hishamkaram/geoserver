package geoserver

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
)

// CoverageStoresService define all geoserver CoverageStores operations
type CoverageStoresService interface {
	GetCoverageStores(workspaceName string) (coverageStores []*Resource, err error)
	GetCoverageStore(workspaceName string, gridName string) (coverageStore *CoverageStore, err error)
	CreateCoverageStore(workspaceName string, coverageStore CoverageStore) (created bool, err error)
	UpdateCoverageStore(workspaceName string, coverageStore CoverageStore) (modified bool, err error)
	DeleteCoverageStore(workspaceName string, coverageStore string, recurse bool) (deleted bool, err error)
}

// CoverageStoresServiceWithContext is the context-aware sibling of [CoverageStoresService].
type CoverageStoresServiceWithContext interface {
	GetCoverageStoresContext(ctx context.Context, workspaceName string) (coverageStores []*Resource, err error)
	GetCoverageStoreContext(ctx context.Context, workspaceName string, gridName string) (coverageStore *CoverageStore, err error)
	CreateCoverageStoreContext(ctx context.Context, workspaceName string, coverageStore CoverageStore) (created bool, err error)
	UpdateCoverageStoreContext(ctx context.Context, workspaceName string, coverageStore CoverageStore) (modified bool, err error)
	DeleteCoverageStoreContext(ctx context.Context, workspaceName string, coverageStore string, recurse bool) (deleted bool, err error)
}

// CoverageStore geoserver coverage store
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

// CoverageStoreRequestBody geoserver coverage store to send to api
type CoverageStoreRequestBody struct {
	CoverageStore *CoverageStore `json:"coverageStore,omitempty"`
}

// GetCoverageStores returns all coverage stores in workspaceName as resources
// using context.Background.
func (g *GeoServer) GetCoverageStores(workspaceName string) (coverageStores []*Resource, err error) {
	return g.GetCoverageStoresContext(context.Background(), workspaceName)
}

// GetCoverageStoresContext is the context-aware variant of [GeoServer.GetCoverageStores].
func (g *GeoServer) GetCoverageStoresContext(ctx context.Context, workspaceName string) (coverageStores []*Resource, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		coverageStores = nil
		err = g.GetError(responseCode, response)
		return
	}
	var coverageStoresResponse struct {
		CoverageStores struct {
			CoverageStore []*Resource `json:"coverageStore,omitempty"`
		} `json:"coverageStores,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &coverageStoresResponse); err != nil {
		return nil, err
	}
	coverageStores = coverageStoresResponse.CoverageStores.CoverageStore
	return
}

// GetCoverageStore returns a single coverage store using context.Background.
func (g *GeoServer) GetCoverageStore(workspaceName string, gridName string) (coverageStore *CoverageStore, err error) {
	return g.GetCoverageStoreContext(context.Background(), workspaceName, gridName)
}

// GetCoverageStoreContext is the context-aware variant of [GeoServer.GetCoverageStore].
func (g *GeoServer) GetCoverageStoreContext(ctx context.Context, workspaceName string, gridName string) (coverageStore *CoverageStore, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", gridName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		coverageStore = nil
		err = g.GetError(responseCode, response)
		return
	}
	var coverageStoreResponse struct {
		CoverageStore *CoverageStore `json:"coverageStore,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &coverageStoreResponse); err != nil {
		return nil, err
	}
	coverageStore = coverageStoreResponse.CoverageStore
	return
}

// CreateCoverageStore creates a coverage store using context.Background.
func (g *GeoServer) CreateCoverageStore(workspaceName string, coverageStore CoverageStore) (created bool, err error) {
	return g.CreateCoverageStoreContext(context.Background(), workspaceName, coverageStore)
}

// CreateCoverageStoreContext is the context-aware variant of [GeoServer.CreateCoverageStore].
func (g *GeoServer) CreateCoverageStoreContext(ctx context.Context, workspaceName string, coverageStore CoverageStore) (created bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores")
	data := CoverageStoreRequestBody{
		CoverageStore: &coverageStore,
	}
	serializedData, serErr := g.SerializeStruct(data)
	if serErr != nil {
		return false, fmt.Errorf("CreateCoverageStore: serialize store: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Data:     bytes.NewBuffer(serializedData),
		DataType: jsonType + "; charset=utf-8",
		Accept:   jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// UpdateCoverageStore performs a partial update on a coverage store using context.Background.
func (g *GeoServer) UpdateCoverageStore(workspaceName string, coverageStore CoverageStore) (modified bool, err error) {
	return g.UpdateCoverageStoreContext(context.Background(), workspaceName, coverageStore)
}

// UpdateCoverageStoreContext is the context-aware variant of [GeoServer.UpdateCoverageStore].
func (g *GeoServer) UpdateCoverageStoreContext(ctx context.Context, workspaceName string, coverageStore CoverageStore) (modified bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStore.Name)
	data := CoverageStoreRequestBody{CoverageStore: &coverageStore}
	serializedData, serErr := g.SerializeStruct(data)
	if serErr != nil {
		return false, fmt.Errorf("UpdateCoverageStore: serialize store: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Data:     bytes.NewBuffer(serializedData),
		DataType: jsonType,
		Accept:   jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		modified = false
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

// DeleteCoverageStore deletes a coverage store using context.Background.
func (g *GeoServer) DeleteCoverageStore(workspaceName string, coverageStore string, recurse bool) (deleted bool, err error) {
	return g.DeleteCoverageStoreContext(context.Background(), workspaceName, coverageStore, recurse)
}

// DeleteCoverageStoreContext is the context-aware variant of [GeoServer.DeleteCoverageStore].
func (g *GeoServer) DeleteCoverageStoreContext(ctx context.Context, workspaceName string, coverageStore string, recurse bool) (deleted bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStore)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
