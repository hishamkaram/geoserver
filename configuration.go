package geoserver

import "bytes"

// ConfigurationService define geoserver Configuration operations
type ConfigurationService interface {
	RestConfigrationCache() (success bool, err error)
	ReloadConfigration() (success bool, err error)
}

//RestConfigrationCache Resets all store, raster, and schema caches.
//This operation is used to force GeoServer to drop all caches and store connections and reconnect to each of them the next time they are needed by a request.
//This is useful in case the stores themselves cache some information about the data structures they manage that may have changed in the meantime.
func (g *GeoServer) RestConfigrationCache() (success bool, err error) {
	targetURL := g.ParseURL("rest", "reset")
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer([]byte("")),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		success = false
		err = g.GetError(responseCode, response)
		return
	}
	success = true
	return

}

//ReloadConfigration Reloads the GeoServer catalog and configuration from disk.
//This operation is used in cases where an external tool has modified the on-disk configuration.
//This operation will also force GeoServer to drop any internal caches and reconnect to all data stores.
func (g *GeoServer) ReloadConfigration() (success bool, err error) {
	targetURL := g.ParseURL("rest", "reload")
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer([]byte("")),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		success = false
		err = g.GetError(responseCode, response)
		return
	}
	success = true
	return
}
