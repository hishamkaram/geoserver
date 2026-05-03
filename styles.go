package geoserver

import (
	"bytes"
	"context"
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

// StyleServiceWithContext is the context-aware sibling of [StyleService].
type StyleServiceWithContext interface {
	GetStylesContext(ctx context.Context, workspaceName string) (styles []*Resource, err error)
	CreateStyleContext(ctx context.Context, workspaceName string, styleName string) (created bool, err error)
	UploadStyleContext(ctx context.Context, data io.Reader, workspaceName string, styleName string, overwrite bool) (success bool, err error)
	DeleteStyleContext(ctx context.Context, workspaceName string, styleName string, purge bool) (deleted bool, err error)
	GetStyleContext(ctx context.Context, workspaceName string, styleName string) (style *Style, err error)
	StyleExistsContext(ctx context.Context, workspaceName string, styleName string) (exists bool, err error)
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

// GetStyles lists styles using context.Background.
func (g *GeoServer) GetStyles(workspaceName string) (styles []*Resource, err error) {
	return g.GetStylesContext(context.Background(), workspaceName)
}

// GetStylesContext is the context-aware variant of [GeoServer.GetStyles].
func (g *GeoServer) GetStylesContext(ctx context.Context, workspaceName string) (styles []*Resource, err error) {
	targetURL := g.stylesURL(workspaceName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
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

// GetStyle fetches a style using context.Background.
func (g *GeoServer) GetStyle(workspaceName string, styleName string) (style *Style, err error) {
	return g.GetStyleContext(context.Background(), workspaceName, styleName)
}

// GetStyleContext is the context-aware variant of [GeoServer.GetStyle].
func (g *GeoServer) GetStyleContext(ctx context.Context, workspaceName string, styleName string) (style *Style, err error) {
	targetURL := g.stylesURL(workspaceName, styleName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
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

// StyleExists checks for a style using context.Background.
func (g *GeoServer) StyleExists(workspaceName string, styleName string) (exists bool, err error) {
	return g.StyleExistsContext(context.Background(), workspaceName, styleName)
}

// StyleExistsContext is the context-aware variant of [GeoServer.StyleExists].
func (g *GeoServer) StyleExistsContext(ctx context.Context, workspaceName string, styleName string) (exists bool, err error) {
	_, styleErr := g.GetStyleContext(ctx, workspaceName, styleName)
	if styleErr != nil {
		return false, styleErr
	}
	return true, nil
}

// CreateStyle creates an empty SLD using context.Background.
func (g *GeoServer) CreateStyle(workspaceName string, styleName string) (created bool, err error) {
	return g.CreateStyleContext(context.Background(), workspaceName, styleName)
}

// CreateStyleContext is the context-aware variant of [GeoServer.CreateStyle].
func (g *GeoServer) CreateStyleContext(ctx context.Context, workspaceName string, styleName string) (created bool, err error) {
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

// UploadStyle uploads SLD content using context.Background.
func (g *GeoServer) UploadStyle(data io.Reader, workspaceName string, styleName string, overwrite bool) (success bool, err error) {
	return g.UploadStyleContext(context.Background(), data, workspaceName, styleName, overwrite)
}

// UploadStyleContext is the context-aware variant of [GeoServer.UploadStyle].
func (g *GeoServer) UploadStyleContext(ctx context.Context, data io.Reader, workspaceName string, styleName string, overwrite bool) (success bool, err error) {
	targetURL := g.stylesURL(workspaceName, styleName)
	exists, existsErr := g.StyleExistsContext(ctx, workspaceName, styleName)
	if existsErr != nil {
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
		created, uploadErr := g.CreateStyleContext(ctx, workspaceName, styleName)
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
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		success = false
		err = g.GetError(responseCode, response)
		return
	}
	success = true
	return
}

// DeleteStyle deletes a style using context.Background.
func (g *GeoServer) DeleteStyle(workspaceName string, styleName string, purge bool) (deleted bool, err error) {
	return g.DeleteStyleContext(context.Background(), workspaceName, styleName, purge)
}

// DeleteStyleContext is the context-aware variant of [GeoServer.DeleteStyle].
func (g *GeoServer) DeleteStyleContext(ctx context.Context, workspaceName string, styleName string, purge bool) (deleted bool, err error) {
	targetURL := g.stylesURL(workspaceName, styleName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"purge": strconv.FormatBool(purge)},
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
