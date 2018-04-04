package geoserver

import "fmt"

// AboutService define all geoserver About operations
type AboutService interface {
	//IsRunning check if geoserver is running return true and error if if error occure
	IsRunning() (running bool, err error)
}

//IsRunning check if geoserver is running
//return true if geoserver running
//and false if not runnging,
//err is an error if error occurredÎ
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
	return
}
