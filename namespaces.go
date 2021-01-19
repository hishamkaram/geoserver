package geoserver

import (
	"bytes"
)

// NamespaceService define all geoserver namespace operations
type NamespaceService interface {

	// NamespaceExists check if Namespace in geoserver or not else return error
	NamespaceExists(Prefix string) (exists bool, err error)

	// GetNamespaces get geoserver Namespaces else return error
	GetNamespaces() (namespaces []*Namespace, err error)

	// GetNamespace get geoserver Namespaces else return error
	GetNamespace(Prefix string) (namespace Namespace, err error)

	// CreateNamespace creates a Namespace else return error
	CreateNamespace(Prefix string, URI string) (created bool, err error)

	//DeleteNamespace delete geoserver Namespace and its reources else return error
	DeleteNamespace(Prefix string) (deleted bool, err error)
}

//Namespace is the Namespace Object
type Namespace struct {
	Prefix   string `json:"prefix,omitempty"`
	URI      string `json:"uri,omitempty"`
	Href     string `json:"href,omitempty"`
	Isolated bool   `json:"isolated,omitempty"`
}

//NamespaceRequestBody is the api body
type NamespaceRequestBody struct {
	Namespace *Namespace `json:"namespace,omitempty"`
}

// CreateNamespace creates a Namespace and return if created or not else return error
func (g *GeoServer) CreateNamespace(Prefix string, URI string) (created bool, err error) {
	//TODO: check if Namespace exist before creating it
	var Namespace = Namespace{Prefix: Prefix, URI: URI}
	serializedNamespace, _ := g.SerializeStruct(NamespaceRequestBody{Namespace: &Namespace})
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
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusCreated {
		//g.logger.Warn(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// NamespaceExists check if Namespace in geoserver or not else return error
func (g *GeoServer) NamespaceExists(Prefix string) (exists bool, err error) {
	_, NamespaceErr := g.GetNamespace(Prefix)
	if NamespaceErr != nil {
		exists = false
		err = NamespaceErr
		return
	}
	exists = true
	return
}

//DeleteNamespace delete geoserver Namespace and its reources else return error
func (g *GeoServer) DeleteNamespace(Prefix string) (deleted bool, err error) {
	url := g.ParseURL("rest", "namespaces", Prefix)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    url,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		//g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}

// GetNamespaces get geoserver namespaces else return error
func (g *GeoServer) GetNamespaces() (namespaces []*Namespace, err error) {
	url := g.ParseURL("rest", "namespaces")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		//g.logger.Warn(string(response))
		namespaces = nil
		err = g.GetError(responseCode, response)
		return
	}
	var NamespaceResponse struct {
		Namespaces struct {
			Namespace []*Namespace
		}
	}
	g.DeSerializeJSON(response, &NamespaceResponse)
	namespaces = NamespaceResponse.Namespaces.Namespace
	return
}

// GetNamespace get geoserver Namespace else return error
func (g *GeoServer) GetNamespace(Prefix string) (namespace Namespace, err error) {
	url := g.ParseURL("rest", "namespaces", Prefix)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		//g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}
	NamespaceResponse := NamespaceRequestBody{
		Namespace: &Namespace{},
	}
	g.DeSerializeJSON(response, &NamespaceResponse)
	namespace = *NamespaceResponse.Namespace
	return
}
