package geoserver

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// StyleService define all geoserver style operations
type StyleService interface {

	// GetStyles
	GetStyles() (styles []Resource, statusCode int)

	//CreateStyle create geoserver sld
	CreateStyle(styleName string) (created bool, statusCode int)

	//UploadStyle upload geoserver sld
	UploadStyle(data io.Reader, styleName string) (success bool, statusCode int)

	//DeleteStyle delete geoserver style
	DeleteStyle(styleName string, purge bool) (deleted bool, statusCode int)
}

//Style holds geoserver style
type Style struct {
	Name            string `json:",omitempty"`
	Format          string `json:",omitempty"`
	Filename        string `json:",omitempty"`
	LanguageVersion struct {
		Version string `json:",omitempty"`
	} `json:",omitempty"`
}

//StyleBody is the api body
type StyleBody struct {
	Style Style `json:"style,omitempty"`
}

// Styles holds a list of geoserver styles
type Styles struct {
	Style []Style `json:",omitempty"`
}

//GetStyles return list of geoserver styles
func (g *GeoServer) GetStyles() (styles []Resource, statusCode int) {
	targetURL := fmt.Sprintf("%srest/styles", g.ServerURL)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		styles = nil
		return
	}
	var stylesResponse struct {
		Style []Resource `json:",omitempty"`
	}
	g.DeSerializeJSON(response, &stylesResponse)
	styles = stylesResponse.Style
	return
}

//CreateStyle create geoserver sld
func (g *GeoServer) CreateStyle(styleName string) (created bool, statusCode int) {
	targetURL := fmt.Sprintf("%srest/styles", g.ServerURL)
	var style = Style{Name: styleName, Filename: styleName + ".sld"}
	serializedStyle, _ := g.SerializeStruct(StyleBody{Style: style})
	xml := bytes.NewBuffer(serializedStyle)
	response, responseCode := g.DoPost(targetURL, xml, jsonType, jsonType)
	statusCode = responseCode
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		return
	}
	created = true
	return
}

//UploadStyle upload geoserver sld
func (g *GeoServer) UploadStyle(data io.Reader, styleName string) (success bool, statusCode int) {
	targetURL := fmt.Sprintf("%srest/styles/%s", g.ServerURL, styleName)
	response, responseCode := g.DoPut(targetURL, data, sldType, jsonType)
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		success = false
		return
	}
	success = true
	return
}

//DeleteStyle delete geoserver style
func (g *GeoServer) DeleteStyle(styleName string, purge bool) (deleted bool, statusCode int) {
	url := fmt.Sprintf("%s/rest/styles/%s", g.ServerURL, styleName)
	response, responseCode := g.DoDelete(url, jsonType, map[string]string{"purge": strconv.FormatBool(purge)})
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		return
	}
	deleted = true
	return
}
