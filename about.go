package geoserver

import "fmt"

// AboutService define all geoserver About operations
type AboutService interface {
	//IsRunning check if geoserver is running return true and statusCode of request
	IsRunning() (running bool, err error)
}

// IsRunning check if geoserver is running
func (g *GeoServer) IsRunning() (running bool, err error) {
	url := fmt.Sprintf("%srest/about/version", g.ServerURL)
	response, responseCode := g.DoGet(url, jsonType, nil)
	if responseCode != statusOk {
		err = statusErrorMapping[responseCode]
		g.logger.Warn(string(response))
		running = false
		return
	}
	running = true
	err = nil
	return
}
