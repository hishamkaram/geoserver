package geoserver

import "fmt"

// AboutService define all geoserver About operations
type AboutService interface {
	//IsRunning check if geoserver is running return true and statusCode of request
	IsRunning() (running bool, statusCode int)
}

// IsRunning check if geoserver is running
func (g *GeoServer) IsRunning() (running bool, statusCode int) {
	url := fmt.Sprintf("%srest/about/version", g.ServerURL)
	_, responseCode := g.DoGet(url, jsonType, nil)
	if responseCode != statusOk {
		running = false
		statusCode = responseCode
		return
	}
	running = true
	statusCode = responseCode
	return
}
