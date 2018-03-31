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
	GetStyles(workspaceName string) (styles []Resource, statusCode int)

	//CreateStyle create geoserver sld
	CreateStyle(workspaceName string, styleName string) (created bool, statusCode int)

	//UploadStyle upload geoserver sld
	UploadStyle(data io.Reader, workspaceName string, styleName string) (success bool, statusCode int)

	//DeleteStyle delete geoserver style
	DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, statusCode int)

	//GetStyle return specific of geoserver style
	GetStyle(workspaceName string, styleName string) (style Style, statusCode int)
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
func (g *GeoServer) GetStyles(workspaceName string) (styles []Resource, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := fmt.Sprintf("%srest/%sstyles", g.ServerURL, workspaceName)
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

//GetStyle return specific of geoserver style
func (g *GeoServer) GetStyle(workspaceName string, styleName string) (style Style, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := fmt.Sprintf("%srest/%sstyles/%s", g.ServerURL, workspaceName, styleName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		style = Style{}
		return
	}
	var stylesResponse StyleBody
	g.DeSerializeJSON(response, &stylesResponse)
	style = stylesResponse.Style
	return
}

//CreateStyle create geoserver sld
func (g *GeoServer) CreateStyle(workspaceName string, styleName string) (created bool, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := fmt.Sprintf("%srest/%sstyles", g.ServerURL, workspaceName)
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
func (g *GeoServer) UploadStyle(data io.Reader, workspaceName string, styleName string) (success bool, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := fmt.Sprintf("%srest/%sstyles/%s", g.ServerURL, workspaceName, styleName)
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
func (g *GeoServer) DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, statusCode int) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	url := fmt.Sprintf("%s/rest/%sstyles/%s", g.ServerURL, workspaceName, styleName)
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
