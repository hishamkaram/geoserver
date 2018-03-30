package geoserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

//Style holds geoserver style
type Style struct {
	Name     string `json:",omitempty"`
	Href     string `json:",omitempty"`
	Filename string `json:",omitempty"`
}

// Styles holds a list of geoserver styles
type Styles struct {
	Style []Style `json:",omitempty"`
}

//GetStyles return list of geoserver styles
func (g *GeoServer) GetStyles() (styles []Style, statusCode int) {
	targetURL := fmt.Sprintf("%srest/styles", g.ServerURL)
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		styles = nil
		return
	}
	var stylesResponse Styles
	err := json.Unmarshal(response, &stylesResponse)
	if err != nil {
		panic(err)
	}
	styles = stylesResponse.Style
	return
}

//CreateStyle create geoserver sld
func (g *GeoServer) CreateStyle(styleName string) (created bool, statusCode int) {
	targetURL := fmt.Sprintf("%srest/styles", g.ServerURL)
	xml := bytes.NewBuffer([]byte(fmt.Sprintf(`<style><name>%s</name><filename>%s.sld</filename></style>`,
		styleName, styleName)))
	_, responseCode := g.DoPost(targetURL, xml, xmlType, jsonType)
	statusCode = responseCode
	if responseCode != statusOk {
		created = false
		return
	}
	created = true
	return
}

//UploadStyle upload geoserver sld
func (g *GeoServer) UploadStyle(data *io.Reader, styleName string) (style Style, statusCode int) {
	targetURL := fmt.Sprintf("%srest/styles/%s", g.ServerURL, styleName)
	response, responseCode := g.DoPost(targetURL, *data, sldType, jsonType)
	statusCode = responseCode
	if responseCode != statusOk {
		style = Style{}
		return
	}
	var stylesResponse Style
	err := json.Unmarshal(response, &stylesResponse)
	if err != nil {
		panic(err)
	}
	style = stylesResponse
	return
}

//DeleteStyle delete geoserver style
func (g *GeoServer) DeleteStyle(styleName string, purge bool) (deleted bool, statusCode int) {
	url := fmt.Sprintf("%s/rest/styles/%s", g.ServerURL, styleName)
	_, responseCode := g.DoDelete(url, jsonType, map[string]string{"purge": strconv.FormatBool(purge)})
	statusCode = responseCode
	if responseCode != statusOk {
		deleted = false
		return
	}
	deleted = true
	return
}
