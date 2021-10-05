package geoserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

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
	//Metadata               *Metadata          `json:"metadata,omitempty"`  //need to fix an implementation due to json parse error
	//SupportedFormats       []string			  `json:"supportedFormats,omitempty"`  //need to fix an implementation due to json parse error
}

type publishedCoverageDescr struct {
	Name               string `json:"name,omitempty"`
	NativeCoverageName string `json:"nativeCoverageName,omitempty"`
}

type publishCoverageRequest struct {
	CoverageDescr *publishedCoverageDescr `json:"coverage,omitempty"`
}

// GetCoverages returns all published raster layers (coverages) for workspace as resources,
// err is an error if error occurred else err is nil
func (g *GeoServer) GetCoverages(workspaceName string) (coverages []*Resource, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coverages")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
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
			return nil, fmt.Errorf("can't parse the coverage data, %v", err)
		} else {
			return []*Resource{}, nil
		}
	}

	return coveragesResponse.Coverages.Coverage, nil
}

// GetStoreCoverages returns all coverages (raster layers) names including unpublished for coverageStore as string list,
// err is an error if error occurred else err is nil
func (g *GeoServer) GetStoreCoverages(workspaceName string, coverageStore string) (coverages []string, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", coverageStore, "coverages")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"list": "all"},
	}
	response, responseCode := g.DoRequest(httpRequest)
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
		return nil, fmt.Errorf("can't parse the coverages data, %v", err)
	}

	return coveragesResponse.List.CoverageName, nil
}

// GetCoverage returns the coverage with name coverageName
// err is an error if error occurred else err is nil
func (g *GeoServer) GetCoverage(workspaceName string, coverageName string) (coverage *Coverage, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coverages", coverageName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	var coverageResponse struct {
		Coverage Coverage
	}

	if err = g.DeSerializeJSON(response, &coverageResponse); err != nil {
		return nil, fmt.Errorf("can't parse the coverage data, %v", err)
	}

	return &coverageResponse.Coverage, nil
}

//DeleteCoverage removes the coverage,
func (g *GeoServer) DeleteCoverage(workspaceName string, layerName string, recurse bool) (deleted bool, err error) {
	//it's just a wrapper about DeleteLayer function as it does the same in the most use cases
	return g.DeleteLayer(workspaceName, layerName, recurse)
}

//UpdateCoverage updates geoserver coverage (raster layer), else returns error,
func (g *GeoServer) UpdateCoverage(workspaceName string, coverage *Coverage) (modified bool, err error) {

	items := strings.Split(coverage.Store.Name, ":")
	if len(items) != 2 {
		return false, errors.New("internal error during coverage update, can't build store name")
	}
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "coveragestores", items[1], "coverages", coverage.Name)

	type coverageUpdateRequestBody struct {
		Coverage Coverage `json:"coverage,omitempty"`
	}

	data := coverageUpdateRequestBody{Coverage: *coverage}

	serializedLayer, _ := g.SerializeStruct(data)
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

//PublishCoverage publishes coverage from coverageStore
func (g *GeoServer) PublishCoverage(workspaceName string, coverageStoreName string, coverageName string) (published bool, err error) {

	publishRequest := publishCoverageRequest{
		&publishedCoverageDescr{
			Name:               coverageName,
			NativeCoverageName: coverageName,
		},
	}
	return g.publishCoverage(workspaceName, coverageStoreName, publishRequest)
}

func (g *GeoServer) publishCoverage(workspaceName string, coverageStoreName string, publishCoverageRequest publishCoverageRequest) (published bool, err error) {

	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "coveragestores", coverageStoreName, "/coverages")

	serializedLayer, _ := g.SerializeStruct(publishCoverageRequest)

	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	return true, nil
}

//PublishGeoTiffLayer publishes geotiff to geoserver
func (g *GeoServer) PublishGeoTiffLayer(workspaceName string, coverageStoreName string, publishName string, fileName string) (published bool, err error) {
	//it was moved from layers.go because this is the better place for raster layers functions (coverages)
	//I tried to maintain the original behavior for backward compatibilities,
	//but it didn't seem to be working as expected from scratch
	//there were no tests for this function and I couldn't reproduce the working case
	publishRequest := publishCoverageRequest{
		&publishedCoverageDescr{
			Name:               publishName,
			NativeCoverageName: fileName,
		},
	}

	return g.publishCoverage(workspaceName, coverageStoreName, publishRequest)
}
