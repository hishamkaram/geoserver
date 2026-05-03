package geoserver

import (
	"bytes"
	"context"
	"fmt"
)

// NamespaceService define all geoserver namespace operations
type NamespaceService interface {

	// NamespaceExists check if Namespace in geoserver or not else return error
	NamespaceExists(prefix string) (exists bool, err error)

	// GetNamespaces get geoserver Namespaces else return error
	GetNamespaces() (namespaces []*Namespace, err error)

	// GetNamespace get geoserver Namespaces else return error
	GetNamespace(prefix string) (namespace Namespace, err error)

	// CreateNamespace creates a Namespace else return error
	CreateNamespace(prefix string, uri string) (created bool, err error)

	// DeleteNamespace delete geoserver Namespace and its reources else return error
	DeleteNamespace(prefix string) (deleted bool, err error)
}

// NamespaceServiceWithContext is the context-aware sibling of [NamespaceService].
type NamespaceServiceWithContext interface {
	NamespaceExistsContext(ctx context.Context, prefix string) (exists bool, err error)
	GetNamespacesContext(ctx context.Context) (namespaces []*Namespace, err error)
	GetNamespaceContext(ctx context.Context, prefix string) (namespace Namespace, err error)
	CreateNamespaceContext(ctx context.Context, prefix string, uri string) (created bool, err error)
	DeleteNamespaceContext(ctx context.Context, prefix string) (deleted bool, err error)
}

// Namespace is the Namespace Object
type Namespace struct {
	Prefix   string `json:"prefix,omitempty"`
	URI      string `json:"uri,omitempty"`
	Href     string `json:"href,omitempty"`
	Isolated bool   `json:"isolated,omitempty"`
}

// NamespaceRequestBody is the api body
type NamespaceRequestBody struct {
	Namespace *Namespace `json:"namespace,omitempty"`
}

// CreateNamespace creates a namespace using context.Background.
func (g *GeoServer) CreateNamespace(prefix string, uri string) (created bool, err error) {
	return g.CreateNamespaceContext(context.Background(), prefix, uri)
}

// CreateNamespaceContext is the context-aware variant of [GeoServer.CreateNamespace].
func (g *GeoServer) CreateNamespaceContext(ctx context.Context, prefix string, uri string) (created bool, err error) {
	//TODO: check if Namespace exist before creating it
	Namespace := Namespace{Prefix: prefix, URI: uri}
	serializedNamespace, serErr := g.SerializeStruct(NamespaceRequestBody{Namespace: &Namespace})
	if serErr != nil {
		return false, fmt.Errorf("CreateNamespace: serialize namespace: %w", serErr)
	}
	targetURL := g.ParseURL("rest", "namespaces")
	data := bytes.NewBuffer(serializedNamespace)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     data,
		DataType: jsonType + "; charset=utf-8",
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// NamespaceExists checks for a namespace using context.Background.
func (g *GeoServer) NamespaceExists(prefix string) (exists bool, err error) {
	return g.NamespaceExistsContext(context.Background(), prefix)
}

// NamespaceExistsContext is the context-aware variant of [GeoServer.NamespaceExists].
func (g *GeoServer) NamespaceExistsContext(ctx context.Context, prefix string) (exists bool, err error) {
	_, NamespaceErr := g.GetNamespaceContext(ctx, prefix)
	if NamespaceErr != nil {
		return false, NamespaceErr
	}
	return true, nil
}

// DeleteNamespace deletes a namespace using context.Background.
func (g *GeoServer) DeleteNamespace(prefix string) (deleted bool, err error) {
	return g.DeleteNamespaceContext(context.Background(), prefix)
}

// DeleteNamespaceContext is the context-aware variant of [GeoServer.DeleteNamespace].
func (g *GeoServer) DeleteNamespaceContext(ctx context.Context, prefix string) (deleted bool, err error) {
	url := g.ParseURL("rest", "namespaces", prefix)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    url,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}

// GetNamespaces lists namespaces using context.Background.
func (g *GeoServer) GetNamespaces() (namespaces []*Namespace, err error) {
	return g.GetNamespacesContext(context.Background())
}

// GetNamespacesContext is the context-aware variant of [GeoServer.GetNamespaces].
func (g *GeoServer) GetNamespacesContext(ctx context.Context) (namespaces []*Namespace, err error) {
	url := g.ParseURL("rest", "namespaces")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		namespaces = nil
		err = g.GetError(responseCode, response)
		return
	}
	var NamespaceResponse struct {
		Namespaces struct {
			Namespace []*Namespace
		}
	}
	if err = g.DeSerializeJSON(response, &NamespaceResponse); err != nil {
		return nil, err
	}
	namespaces = NamespaceResponse.Namespaces.Namespace
	return
}

// GetNamespace fetches a namespace by prefix using context.Background.
func (g *GeoServer) GetNamespace(prefix string) (namespace Namespace, err error) {
	return g.GetNamespaceContext(context.Background(), prefix)
}

// GetNamespaceContext is the context-aware variant of [GeoServer.GetNamespace].
func (g *GeoServer) GetNamespaceContext(ctx context.Context, prefix string) (namespace Namespace, err error) {
	url := g.ParseURL("rest", "namespaces", prefix)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}
	NamespaceResponse := NamespaceRequestBody{
		Namespace: &Namespace{},
	}
	if err = g.DeSerializeJSON(response, &NamespaceResponse); err != nil {
		return Namespace{}, err
	}
	namespace = *NamespaceResponse.Namespace
	return
}
