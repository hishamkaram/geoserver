package geoserver

import "fmt"

//Workspace is the Workspace Object
type Workspace struct {
	Name string
	Href string
}

//CreateWorkspace function to create current geoserver struct workspace
func (g *GeoServer) CreateWorkspace() ([]byte, int) {
	//TODO: check if workspace exist before creating it
	var xml = fmt.Sprintf("<workspace><name>%s</name></workspace>", g.WorkspaceName)
	var targetURL = fmt.Sprintf("%srest/workspaces", g.ServerURL)
	data := []byte(xml)
	return g.DoPost(targetURL, data, "text/xml")
}
