package geoserver

import (
	"github.com/hishamkaram/geoserver/wms"
)

// GetCapabilities retrieves metadata about the WMS service, including supported
// operations and parameters and a list of the available layers.
func (g *GeoServer) GetCapabilities(workspaceName string) (caps *wms.Capabilities, err error) {
	targetURL := g.ParseURL(workspaceName, "wms")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: appXMLType,
		URL:    targetURL,
		Query:  map[string]string{"service": "wms", "version": "1.1.1", "request": "GetCapabilities"},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		caps = nil
		err = g.GetError(responseCode, response)
		return
	}
	caps, err = wms.ParseCapabilitiesE(response)
	if err != nil {
		g.logger.Errorf("GetCapabilities: parse: %v", err)
	}
	return
}
