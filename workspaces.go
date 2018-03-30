package geoserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// WorkspaceService define all geoserver workspace operations
type WorkspaceService interface {

	// WorkspaceExists check if workspace in geoserver or not
	WorkspaceExists(workspaceName string) (exists bool, statusCode int)

	// GetWorkspaces get geoserver workspaces
	GetWorkspaces() (workspaces []Resource, statusCode int)

	// CreateWorkspace creates a workspace
	CreateWorkspace(workspaceName string) (created bool, statusCode int)

	// DeleteWorkspace deletes a workspace
	DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, statusCode int)
}

//Workspace is the Workspace Object
type Workspace struct {
	Name string `json:"name,omitempty"`
	Href string `json:"href,omitempty"`
}

//WorkspaceBody is the api body
type WorkspaceBody struct {
	Workspace Workspace `json:"workspace,omitempty"`
}

//CreateWorkspace function to create current geoserver struct workspace
func (g *GeoServer) CreateWorkspace(workspaceName string) (created bool, statusCode int) {
	//TODO: check if workspace exist before creating it
	var workspace = Workspace{Name: workspaceName}
	serializedWorkspace, _ := g.SerializeStruct(WorkspaceBody{Workspace: workspace})
	var targetURL = fmt.Sprintf("%srest/workspaces", g.ServerURL)
	data := bytes.NewBuffer(serializedWorkspace)
	_, responseCode := g.DoPost(targetURL, data, jsonType+"; charset=utf-8", jsonType)
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
func (g *GeoServer) GetWorkspaces() (workspaces []Resource, statusCode int) {
	url := fmt.Sprintf("%srest/workspaces", g.ServerURL)
	response, responseCode := g.DoGet(url, jsonType, nil)
	statusCode = responseCode
	if responseCode != statusOk {
		workspaces = nil
		return
	}
	var workspaceResponse struct {
		Workspaces struct {
			Workspace []Resource
		}
	}
	err := json.Unmarshal(response, &workspaceResponse)
	if err != nil {
		panic(err)
	}
	workspaces = workspaceResponse.Workspaces.Workspace
	return
}
