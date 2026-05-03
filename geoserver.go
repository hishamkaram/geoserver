package geoserver

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

// GeoServer is the configuration object for talking to a GeoServer instance.
//
// NOTE: Fields are exported and mutable for backward compatibility with v1.0.x,
// but the type is NOT safe for concurrent mutation. Construct an instance once
// (via [GetCatalog] or [New]) and treat it as read-only thereafter.
// Concurrent reads against the same instance are safe.
// A concurrency-safe redesign with private fields is planned for v2.
type GeoServer struct {
	WorkspaceName string `yaml:"workspace"`
	ServerURL     string `yaml:"geoserver_url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HttpClient    *http.Client
	logger        *logrus.Logger
}

// LoadConfig load geoserver config from yaml file
func (g *GeoServer) LoadConfig(configFile string) (geoserver *GeoServer, err error) {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		g.logger.Errorf("yamlFile.Get err   %v ", err)
		return
	}
	err = yaml.Unmarshal(yamlFile, g)
	if err != nil {
		g.logger.Errorf("Unmarshal: %v", err)
		return
	}
	g.logger = GetLogger()
	geoserver = g
	return
}

// GetGeoserverRequest creates an HTTP request with GeoServer credentials and
// headers populated.
//
// On failure to construct the request (e.g. an invalid URL), this method logs
// the error and returns nil. Callers that need to distinguish failure should
// use [GeoServer.GetGeoserverRequestE].
func (g *GeoServer) GetGeoserverRequest(
	targetURL string,
	method string,
	accept string,
	data io.Reader,
	contentType string) (request *http.Request) {
	request, err := g.GetGeoserverRequestE(targetURL, method, accept, data, contentType)
	if err != nil {
		g.logger.Errorf("GetGeoserverRequest: %v", err)
		return nil
	}
	return request
}

// GetGeoserverRequestE is like [GeoServer.GetGeoserverRequest] but returns an
// explicit error instead of logging and returning nil. This is the variant new
// code should use; the non-E sibling is preserved for backward compatibility.
func (g *GeoServer) GetGeoserverRequestE(
	targetURL string,
	method string,
	accept string,
	data io.Reader,
	contentType string) (*http.Request, error) {
	// context.Background() here is a placeholder until v1.1's *Context
	// method variants land in A3; per-request cancellation will then flow
	// through GetGeoserverRequestContext, which uses the caller's context.
	request, err := http.NewRequestWithContext(context.Background(), method, targetURL, data)
	if err != nil {
		return nil, err
	}
	if data != nil {
		request.Header.Set(contentTypeHeader, contentType)
	}
	if accept != "" {
		request.Header.Set(acceptHeader, accept)
	}
	request.SetBasicAuth(g.Username, g.Password)
	return request, nil
}
