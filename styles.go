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
	GetStyles(workspaceName string) (styles []*Resource, err error)

	//CreateStyle create geoserver sld
	CreateStyle(workspaceName string, styleName string) (created bool, err error)

	//UploadStyle upload geoserver sld
	UploadStyle(data io.Reader, workspaceName string, styleName string) (success bool, err error)

	//DeleteStyle delete geoserver style
	DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, err error)

	//GetStyle return specific of geoserver style
	GetStyle(workspaceName string, styleName string) (style *Style, err error)
}

//LanguageVersion style version
type LanguageVersion struct {
	Version string `json:"version,omitempty"`
}

//Style holds geoserver style
type Style struct {
	Name            string           `json:"name,omitempty"`
	Format          string           `json:"format,omitempty"`
	Filename        string           `json:"filename,omitempty"`
	LanguageVersion *LanguageVersion `json:"languageVersion,omitempty"`
}

//StyleRequestBody is the api body
type StyleRequestBody struct {
	Style *Style `json:"style,omitempty"`
}

// Styles holds a list of geoserver styles
type Styles struct {
	Style []Style `json:"styles,omitempty"`
}

//GetStyles return list of geoserver styles and err if error occurred,
//if workspace is "" will return non-workspce styles
func (g *GeoServer) GetStyles(workspaceName string) (styles []*Resource, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := fmt.Sprintf("%srest/%sstyles", g.ServerURL, workspaceName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		styles = nil
		err = statusErrorMapping[responseCode]
		return
	}
	var stylesResponse struct {
		Styles struct {
			Style []*Resource `json:"style,omitempty"`
		} `json:"styles,omitempty"`
	}
	g.DeSerializeJSON(response, &stylesResponse)
	styles = stylesResponse.Styles.Style
	return
}

//GetStyle return specific of geoserver style,
//if workspace is "" will return non-workspce styles
func (g *GeoServer) GetStyle(workspaceName string, styleName string) (style *Style, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "styles", styleName)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		style = &Style{}
		err = statusErrorMapping[responseCode]
		return
	}
	var stylesResponse StyleRequestBody
	g.DeSerializeJSON(response, &stylesResponse)
	style = stylesResponse.Style
	return
}

//CreateStyle create geoserver empty sld with name and filename is(${styleName.sld}),
//if workspace is "" will create geoserver public style
func (g *GeoServer) CreateStyle(workspaceName string, styleName string) (created bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "styles")
	var style = Style{Name: styleName, Filename: fmt.Sprintf("%s.sld", styleName)}
	serializedStyle, _ := g.SerializeStruct(StyleRequestBody{Style: &style})
	data := bytes.NewBuffer(serializedStyle)
	response, responseCode := g.DoPost(targetURL, data, jsonType, jsonType)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		err = statusErrorMapping[responseCode]
		return
	}
	created = true
	return
}

//UploadStyle upload geoserver sld,
//if workspace is "" will upload geoserver public style sld , return err if error occurred
func (g *GeoServer) UploadStyle(data io.Reader, workspaceName string, styleName string) (success bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "styles", styleName)
	response, responseCode := g.DoPut(targetURL, data, sldType, jsonType)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		success = false
		err = statusErrorMapping[responseCode]
		return
	}
	success = true
	return
}

//DeleteStyle delete geoserver style,
//if workspace is "" will delete geoserver public style , return err if error occurred
func (g *GeoServer) DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, err error) {
	if workspaceName != "" {
		workspaceName = fmt.Sprintf("workspaces/%s/", workspaceName)
	}
	targetURL := g.ParseURL("rest", workspaceName, "styles", styleName)
	response, responseCode := g.DoDelete(targetURL, jsonType, map[string]string{"purge": strconv.FormatBool(purge)})
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = statusErrorMapping[responseCode]
		return
	}
	deleted = true
	return
}
