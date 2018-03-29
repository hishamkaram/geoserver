package geoserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

//Workspace is the Workspace Object
type Workspace struct {
	Name string
	Href string
}

//WorkspaceResponse respreseent json from api
type WorkspaceResponse struct {
	Workspace []Workspace
}

//CreateWorkspace function to create current geoserver struct workspace
func (g *GeoServer) CreateWorkspace(workspaceName string) (created bool, statusCode int) {
	//TODO: check if workspace exist before creating it
	var xml = fmt.Sprintf("<workspace><name>%s</name></workspace>", workspaceName)
	var targetURL = fmt.Sprintf("%srest/workspaces", g.ServerURL)
	data := bytes.NewBuffer([]byte(xml))
	_, responseCode := g.DoPost(targetURL, data, xmlType, jsonType)
	statusCode = responseCode
	if responseCode != statusCreated {
		created = false
		return
	}
	created = true
	return
}

//WorkspaceExists check if workspace exists in geoserver
func (g *GeoServer) WorkspaceExists(workspaceName string) (exists bool, statusCode int) {
	url := fmt.Sprintf("%s/rest/workspaces/%s", g.ServerURL, workspaceName)
	_, responseCode := g.DoGet(url, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		exists = false
		return
	}
	exists = true
	return
}

//DeleteWorkspace delete geoserver workspace and its reources
func (g *GeoServer) DeleteWorkspace(workspaceName string, recurse bool) (created bool, statusCode int) {
	url := fmt.Sprintf("%s/rest/workspaces/%s", g.ServerURL, workspaceName)
	_, responseCode := g.DoDelete(url, jsonType, map[string]string{"recurse": strconv.FormatBool(recurse)})
	statusCode = responseCode
	if responseCode != statusOk {
		created = false
		return
	}
	created = true
	return
}

//GetWorkspaces  get all geoserver workspaces
func (g *GeoServer) GetWorkspaces() (workspaces []Workspace, statusCode int) {
	url := fmt.Sprintf("%s/rest/workspaces", g.ServerURL)
	response, responseCode := g.DoGet(url, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		workspaces = nil
		return
	}
	var workspaceResponse WorkspaceResponse
	err := json.Unmarshal([]byte(response), &workspaceResponse)
	if err != nil {
		panic(err)
	}

	workspaces = workspaceResponse.Workspace
	return
}

//TODO: ChangeWorkSpace
