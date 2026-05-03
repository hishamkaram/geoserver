package geoserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CoverageService is the interface that bundles GeoServer coverage (raster
// layer) operations on a *GeoServer.
//
// In v1.0 these methods existed on *GeoServer but were not exposed through
// any service interface; v1.1 makes them addressable through the Catalog.
type CoverageService interface {
	GetCoverages(workspaceName string) (coverages []*Resource, err error)
	GetStoreCoverages(workspaceName string, coverageStore string) (coverages []string, err error)
	GetCoverage(workspaceName string, coverageName string) (coverage *Coverage, err error)
	DeleteCoverage(workspaceName string, layerName string, recurse bool) (deleted bool, err error)
	UpdateCoverage(workspaceName string, coverage *Coverage) (modified bool, err error)
	PublishCoverage(workspaceName string, coverageStoreName string, coverageName string, publishName string) (published bool, err error)
	PublishGeoTiffLayer(workspaceName string, coverageStoreName string, publishName string, fileName string) (published bool, err error)
}

// CoverageServiceWithContext is the context-aware sibling of [CoverageService].
type CoverageServiceWithContext interface {
	GetCoveragesContext(ctx context.Context, workspaceName string) (coverages []*Resource, err error)
	GetStoreCoveragesContext(ctx context.Context, workspaceName string, coverageStore string) (coverages []string, err error)
	GetCoverageContext(ctx context.Context, workspaceName string, coverageName string) (coverage *Coverage, err error)
	DeleteCoverageContext(ctx context.Context, workspaceName string, layerName string, recurse bool) (deleted bool, err error)
	UpdateCoverageContext(ctx context.Context, workspaceName string, coverage *Coverage) (modified bool, err error)
	PublishCoverageContext(ctx context.Context, workspaceName string, coverageStoreName string, coverageName string, publishName string) (published bool, err error)
	PublishGeoTiffLayerContext(ctx context.Context, workspaceName string, coverageStoreName string, publishName string, fileName string) (published bool, err error)
}

// Coverage is geoserver Coverage (raster layer) data struct
type Coverage struct {
	Name                 string             `json:"name,omitempty"`
	NativeCoverageName   string             `json:"nativeCoverageName,omitempty"`
	NativeName           string             `json:"nativeName,omitempty"`
	NativeFormat         string             `json:"nativeFormat,omitempty"`
	Namespace            *Resource          `json:"namespace,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Abstract             string             `json:"abstract,omitempty"`
	Keywords             *Keywords          `json:"keywords,omitempty"`
	NativeCRS            *CRSType           `json:"nativeCRS,omitempty"`
	Srs                  string             `json:"srs,omitempty"`
	Enabled              bool               `json:"enabled,omitempty"`
	NativeBoundingBox    *NativeBoundingBox `json:"nativeBoundingBox,omitempty"`
	LatLonBoundingBox    *LatLonBoundingBox `json:"latLonBoundingBox,omitempty"`
	ProjectionPolicy     string             `json:"projectionPolicy,omitempty"`
	Store                *Resource          `json:"store,omitempty"`
	CqlFilter            string             `json:"cqlFilter,omitempty"`
	OverridingServiceSRS bool               `json:"overridingServiceSRS,omitempty"`
	// Metadata               *Metadata          `json:"metadata,omitempty"`  //need to fix the implementation due to json parse error
	// SupportedFormats       []string			  `json:"supportedFormats,omitempty"`  //need to fix the implementation due to json parse error
}

type publishedCoverageDescr struct {
	Name               string `json:"name,omitempty"`
	NativeCoverageName string `json:"nativeCoverageName,omitempty"`
}

type publishCoverageRequest struct {
	CoverageDescr *publishedCoverageDescr `json:"coverage,omitempty"`
}

// GetCoverages lists coverages using context.Background.
func (g *GeoServer) GetCoverages(workspaceName string) (coverages []*Resource, err error) {
	return g.GetCoveragesContext(context.Background(), workspaceName)
}

// GetCoveragesContext is the context-aware variant of [GeoServer.GetCoverages].
func (g *GeoServer) GetCoveragesContext(ctx context.Context, workspaceName string) (coverages []*Resource, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coverages")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	var coveragesResponse struct {
		Coverages struct {
			Coverage []*Resource `json:"coverage,omitempty"`
		} `json:"coverages,omitempty"`
	}

	var coveragesEmptyResponse struct {
		Coverages string
	}

	if err = json.Unmarshal(response, &coveragesResponse); err != nil {
		if err = g.DeSerializeJSON(response, &coveragesEmptyResponse); err != nil {
			return nil, fmt.Errorf("can't parse the coverage data, %w", err)
		}
		return []*Resource{}, nil
	}

	return coveragesResponse.Coverages.Coverage, nil
}

// GetStoreCoverages lists store coverages using context.Background.
func (g *GeoServer) GetStoreCoverages(workspaceName string, coverageStore string) (coverages []string, err error) {
	return g.GetStoreCoveragesContext(context.Background(), workspaceName, coverageStore)
}

// GetStoreCoveragesContext is the context-aware variant of [GeoServer.GetStoreCoverages].
func (g *GeoServer) GetStoreCoveragesContext(ctx context.Context, workspaceName string, coverageStore string) (coverages []string, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStore, "coverages")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"list": "all"},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	var coveragesResponse struct {
		List struct {
			CoverageName []string `json:"string,omitempty"`
		} `json:"list,omitempty"`
	}

	if err = g.DeSerializeJSON(response, &coveragesResponse); err != nil {
		return nil, fmt.Errorf("can't parse the coverages data, %w", err)
	}

	return coveragesResponse.List.CoverageName, nil
}

// GetCoverage fetches a coverage using context.Background.
func (g *GeoServer) GetCoverage(workspaceName string, coverageName string) (coverage *Coverage, err error) {
	return g.GetCoverageContext(context.Background(), workspaceName, coverageName)
}

// GetCoverageContext is the context-aware variant of [GeoServer.GetCoverage].
func (g *GeoServer) GetCoverageContext(ctx context.Context, workspaceName string, coverageName string) (coverage *Coverage, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coverages", coverageName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	var coverageResponse struct {
		Coverage Coverage
	}

	if err = g.DeSerializeJSON(response, &coverageResponse); err != nil {
		return nil, fmt.Errorf("can't parse the coverage data, %w", err)
	}

	return &coverageResponse.Coverage, nil
}

// DeleteCoverage deletes a coverage using context.Background.
func (g *GeoServer) DeleteCoverage(workspaceName string, layerName string, recurse bool) (deleted bool, err error) {
	return g.DeleteCoverageContext(context.Background(), workspaceName, layerName, recurse)
}

// DeleteCoverageContext is the context-aware variant of [GeoServer.DeleteCoverage].
func (g *GeoServer) DeleteCoverageContext(ctx context.Context, workspaceName string, layerName string, recurse bool) (deleted bool, err error) {
	// it's just a wrapper about DeleteLayer function as it does the same in the most use cases
	return g.DeleteLayerContext(ctx, workspaceName, layerName, recurse)
}

// UpdateCoverage updates a coverage using context.Background.
func (g *GeoServer) UpdateCoverage(workspaceName string, coverage *Coverage) (modified bool, err error) {
	return g.UpdateCoverageContext(context.Background(), workspaceName, coverage)
}

// UpdateCoverageContext is the context-aware variant of [GeoServer.UpdateCoverage].
func (g *GeoServer) UpdateCoverageContext(ctx context.Context, workspaceName string, coverage *Coverage) (modified bool, err error) {
	if coverage == nil || coverage.Store == nil {
		return false, errors.New("UpdateCoverage: coverage and coverage.Store must be non-nil")
	}
	items := strings.Split(coverage.Store.Name, ":")
	if len(items) != 2 {
		return false, fmt.Errorf("UpdateCoverage: store name %q is not in the form workspace:store", coverage.Store.Name)
	}
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", items[1], "coverages", coverage.Name)

	type coverageUpdateRequestBody struct {
		Coverage Coverage `json:"coverage,omitempty"`
	}

	data := coverageUpdateRequestBody{Coverage: *coverage}

	serializedLayer, serErr := g.SerializeStruct(data)
	if serErr != nil {
		return false, fmt.Errorf("UpdateCoverage: serialize coverage: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

// PublishCoverage publishes a coverage using context.Background.
func (g *GeoServer) PublishCoverage(workspaceName string, coverageStoreName string, coverageName string, publishName string) (published bool, err error) {
	return g.PublishCoverageContext(context.Background(), workspaceName, coverageStoreName, coverageName, publishName)
}

// PublishCoverageContext is the context-aware variant of [GeoServer.PublishCoverage].
func (g *GeoServer) PublishCoverageContext(ctx context.Context, workspaceName string, coverageStoreName string, coverageName string, publishName string) (published bool, err error) {
	if publishName == "" {
		publishName = coverageName
	}

	publishRequest := publishCoverageRequest{
		&publishedCoverageDescr{
			Name:               publishName,
			NativeCoverageName: coverageName,
		},
	}
	return g.publishCoverage(ctx, workspaceName, coverageStoreName, publishRequest)
}

// publishCoverage publishes coverage to the given workspace's coverage store.
// If workspaceName is empty, the global /rest/coveragestores endpoint is used.
func (g *GeoServer) publishCoverage(ctx context.Context, workspaceName string, coverageStoreName string, request publishCoverageRequest) (published bool, err error) {
	var targetURL string
	if workspaceName == "" {
		targetURL = g.ParseURL("rest", "coveragestores", coverageStoreName, "coverages")
	} else {
		targetURL = g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStoreName, "coverages")
	}

	serializedLayer, serErr := g.SerializeStruct(request)
	if serErr != nil {
		return false, fmt.Errorf("publishCoverage: serialize request: %w", serErr)
	}

	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	return true, nil
}

// PublishGeoTiffLayer publishes a GeoTIFF using context.Background.
func (g *GeoServer) PublishGeoTiffLayer(workspaceName string, coverageStoreName string, publishName string, fileName string) (published bool, err error) {
	return g.PublishGeoTiffLayerContext(context.Background(), workspaceName, coverageStoreName, publishName, fileName)
}

// PublishGeoTiffLayerContext is the context-aware variant of [GeoServer.PublishGeoTiffLayer].
func (g *GeoServer) PublishGeoTiffLayerContext(ctx context.Context, workspaceName string, coverageStoreName string, publishName string, fileName string) (published bool, err error) {
	// it was moved from layers.go because this is the better place for raster layers functions (coverages)
	// I tried to maintain the original behavior for backward compatibilities,
	// but it didn't seem to be working as expected from scratch
	// there were no tests for this function and I couldn't reproduce the working case
	publishRequest := publishCoverageRequest{
		&publishedCoverageDescr{
			Name:               publishName,
			NativeCoverageName: fileName,
		},
	}

	return g.publishCoverage(ctx, workspaceName, coverageStoreName, publishRequest)
}
