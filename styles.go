package geoserver

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// StyleService define all geoserver style operations
type StyleService interface {
	GetStyles(workspaceName string) (styles []*Resource, err error)

	CreateStyle(workspaceName string, styleName string) (created bool, err error)

	UploadStyle(data io.Reader, workspaceName string, styleName string, overwrite bool) (success bool, err error)

	DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, err error)

	GetStyle(workspaceName string, styleName string) (style *Style, err error)

	StyleExists(workspaceName string, styleName string) (exists bool, err error)
}

// LanguageVersion style version
type LanguageVersion struct {
	Version string `json:"version,omitempty"`
}

// Style holds geoserver style
type Style struct {
	Name            string           `json:"name,omitempty"`
	Format          string           `json:"format,omitempty"`
	Filename        string           `json:"filename,omitempty"`
	LanguageVersion *LanguageVersion `json:"languageVersion,omitempty"`
}

// StyleRequestBody is the api body
type StyleRequestBody struct {
	Style *Style `json:"style,omitempty"`
}

// Styles holds a list of geoserver styles
type Styles struct {
	Style []Style `json:"styles,omitempty"`
}

// stylesURL builds the /rest[/workspaces/{ws}]/styles[/{name}] URL with proper
// path-escaping. If workspaceName is empty, the global styles endpoint is used.
func (g *GeoServer) stylesURL(workspaceName string, extra ...string) string {
	parts := []string{"rest"}
	if workspaceName != "" {
		parts = append(parts, "workspaces", workspaceName)
	}
	parts = append(parts, "styles")
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// GetStyles return list of geoserver styles and err if error occurred,
// if workspace is "" will return non-workspce styles
func (g *GeoServer) GetStyles(workspaceName string) (styles []*Resource, err error) {
	targetURL := g.stylesURL(workspaceName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		styles = nil
		err = g.GetError(responseCode, response)
		return
	}
	var stylesResponse struct {
		Styles struct {
			Style []*Resource `json:"style,omitempty"`
		} `json:"styles,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &stylesResponse); err != nil {
		return nil, err
	}
	styles = stylesResponse.Styles.Style
	return
}

// GetStyle return specific of geoserver style,
// if workspace is "" will return non-workspce styles
func (g *GeoServer) GetStyle(workspaceName string, styleName string) (style *Style, err error) {
	targetURL := g.stylesURL(workspaceName, styleName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		style = &Style{}
		err = g.GetError(responseCode, response)
		return
	}
	var stylesResponse StyleRequestBody
	if err = g.DeSerializeJSON(response, &stylesResponse); err != nil {
		return nil, err
	}
	style = stylesResponse.Style
	return
}

// StyleExists return true if style exists in geoserver
func (g *GeoServer) StyleExists(workspaceName string, styleName string) (exists bool, err error) {
	_, styleErr := g.GetStyle(workspaceName, styleName)
	if styleErr != nil {
		exists = false
		err = styleErr
		return
	}
	exists = true
	return
}

// CreateStyle create geoserver empty sld with name and filename is(${styleName.sld}),
// if workspace is "" will create geoserver public style
func (g *GeoServer) CreateStyle(workspaceName string, styleName string) (created bool, err error) {
	targetURL := g.stylesURL(workspaceName)
	style := Style{Name: styleName, Filename: fmt.Sprintf("%s.sld", styleName)}
	serializedStyle, serErr := g.SerializeStruct(StyleRequestBody{Style: &style})
	if serErr != nil {
		return false, fmt.Errorf("CreateStyle: serialize style: %w", serErr)
	}
	data := bytes.NewBuffer(serializedStyle)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     data,
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
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

// UploadStyle upload geoserver sld,
// if workspace is "" will upload geoserver public style sld , return err if error occurred
func (g *GeoServer) UploadStyle(data io.Reader, workspaceName string, styleName string, overwrite bool) (success bool, err error) {
	targetURL := g.stylesURL(workspaceName, styleName)
	exists, existsErr := g.StyleExists(workspaceName, styleName)
	// existsErr is non-nil for any HTTP failure other than the missing-style
	// happy path. Surface it instead of silently ignoring (the prior behavior
	// would race against StyleExists returning the wrong answer on transient
	// failures).
	if existsErr != nil {
		// Distinguish 404 (style absent — proceed) from real errors.
		// Without a typed-error system in v1.0 this was awkward; in v1.1
		// callers can use errors.Is(err, ErrNotFound). For backwards
		// compatibility we keep proceeding when StyleExists reports false
		// (the bool is the source of truth here).
		_ = existsErr
	}
	if exists && !overwrite {
		g.logger.Error(exists)
		success = false
		err = g.GetError(statusForbidden, []byte("Style Already Exists"))
		return
	}
	if !exists {
		created, uploadErr := g.CreateStyle(workspaceName, styleName)
		if !created {
			success = false
			err = uploadErr
			return
		}
	}
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     data,
		DataType: sldType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		success = false
		err = g.GetError(responseCode, response)
		return
	}
	success = true
	return
}

// DeleteStyle delete geoserver style,
// if workspace is "" will delete geoserver public style , return err if error occurred
func (g *GeoServer) DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, err error) {
	targetURL := g.stylesURL(workspaceName, styleName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"purge": strconv.FormatBool(purge)},
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
