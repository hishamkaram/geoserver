package geoserver

// AboutService define all geoserver About operations
type AboutService interface {
	//IsRunning check if geoserver is running return true and error if if error occure
	IsRunning() (running bool, err error)
}

//IsRunning check if geoserver is running \n
//return true if geoserver running,
//and false if not runnging,
//err is an error if error occurredÎ
func (g *GeoServer) IsRunning() (running bool, err error) {
	targetURL := g.ParseURL("rest", "about", "version")
	response, responseCode := g.DoGet(targetURL, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		err = statusErrorMapping[responseCode]
		running = false
		return
	}
	running = true
	return
}
