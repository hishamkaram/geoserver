package geoserver

import "context"

// AboutService define all geoserver About operations
type AboutService interface {
	// IsRunning check if geoserver is running return true and error if if error occure
	IsRunning() (running bool, err error)
}

// AboutServiceWithContext is the context-aware sibling of [AboutService].
type AboutServiceWithContext interface {
	IsRunningContext(ctx context.Context) (running bool, err error)
}

// IsRunning probes /rest/about/version and returns true if GeoServer
// answered with 200 OK. Uses context.Background; see [GeoServer.IsRunningContext].
func (g *GeoServer) IsRunning() (running bool, err error) {
	return g.IsRunningContext(context.Background())
}

// IsRunningContext is the context-aware variant of [GeoServer.IsRunning].
func (g *GeoServer) IsRunningContext(ctx context.Context) (running bool, err error) {
	targetURL := g.ParseURL("rest", "about", "version")
	httpRequest := HTTPRequest{
		URL:    targetURL,
		Method: getMethod,
		Accept: jsonType,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		running = false
		return
	}
	running = true
	return
}
