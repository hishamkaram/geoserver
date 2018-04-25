package geoserver

import (
	"bytes"
	"strconv"
)

// WorkspaceService define all geoserver workspace operations
type WorkspaceService interface {

	// WorkspaceExists check if workspace in geoserver or not else return error
	WorkspaceExists(workspaceName string) (exists bool, err error)

	// GetWorkspaces get geoserver workspaces else return error
	GetWorkspaces() (workspaces []*Resource, err error)

	// CreateWorkspace creates a workspace else return error
	CreateWorkspace(workspaceName string) (created bool, err error)

	//DeleteWorkspace delete geoserver workspace and its reources else return error
	DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, err error)
}

//Workspace is the Workspace Object
type Workspace struct {
	Name           string `json:"name,omitempty"`
	Isolated       bool   `json:"isolated,omitempty"`
	DataStores     string `json:"dataStores,omitempty"`
	CoverageStores string `json:"coverageStores,omitempty"`
	WmsStores      string `json:"wmsStores,omitempty"`
	WmtsStores     string `json:"wmtsStores,omitempty"`
}

//WorkspaceRequestBody is the api body
type WorkspaceRequestBody struct {
	Workspace *Workspace `json:"workspace,omitempty"`
}

// CreateWorkspace creates a workspace and return if created or not else return error
func (g *GeoServer) CreateWorkspace(workspaceName string) (created bool, err error) {
	//TODO: check if workspace exist before creating it
	var workspace = Workspace{Name: workspaceName}
	serializedWorkspace, _ := g.SerializeStruct(WorkspaceRequestBody{Workspace: &workspace})
	targetURL := g.ParseURL("rest", "workspaces")
	data := bytes.NewBuffer(serializedWorkspace)
	response, responseCode := g.DoPost(targetURL, data, jsonType+"; charset=utf-8", jsonType)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		err = statusErrorMapping[responseCode]
		return
	}
	created = true
	return
}

// WorkspaceExists check if workspace in geoserver or not else return error
func (g *GeoServer) WorkspaceExists(workspaceName string) (exists bool, err error) {
	url := g.ParseURL("rest", "workspaces", workspaceName)
	response, responseCode := g.DoGet(url, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		exists = false
		err = statusErrorMapping[responseCode]
		return
	}
	exists = true
	return
}

//DeleteWorkspace delete geoserver workspace and its reources else return error
func (g *GeoServer) DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, err error) {
	url := g.ParseURL("rest", "workspaces", workspaceName)
	response, responseCode := g.DoDelete(url, jsonType, map[string]string{"recurse": strconv.FormatBool(recurse)})
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = statusErrorMapping[responseCode]
		return
	}
	deleted = true
	return
}

// GetWorkspaces get geoserver workspaces else return error
func (g *GeoServer) GetWorkspaces() (workspaces []*Resource, err error) {
	url := g.ParseURL("rest", "workspaces")
	response, responseCode := g.DoGet(url, jsonType, nil)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		workspaces = nil
		err = statusErrorMapping[responseCode]
		return
	}
	var workspaceResponse struct {
		Workspaces struct {
			Workspace []*Resource
		}
	}
	g.DeSerializeJSON(response, &workspaceResponse)
	workspaces = workspaceResponse.Workspaces.Workspace
	return
}
