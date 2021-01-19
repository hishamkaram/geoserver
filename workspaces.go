package geoserver

import (
	"bytes"
	"net/http"
	"strconv"
)

// WorkspaceService define all geoserver workspace operations
type WorkspaceService interface {

	// WorkspaceExists check if workspace in geoserver or not else return error
	WorkspaceExists(workspaceName string) (exists bool, err error)

	// GetWorkspaces get geoserver workspaces else return error
	GetWorkspaces() (workspaces []*Resource, err error)

	// GetWorkspace get geoserver workspaces else return error
	GetWorkspace(workspaceName string) (workspace Workspace, err error)

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
	httpRequest := HTTPRequest{
		Method:   http.MethodPost,
		Accept:   jsonType,
		Data:     data,
		DataType: jsonType + "; charset=utf-8",
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusCreated {
		//g.logger.Warn(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// WorkspaceExists check if workspace in geoserver or not else return error
func (g *GeoServer) WorkspaceExists(workspaceName string) (exists bool, err error) {
	_, workspaceErr := g.GetWorkspace(workspaceName)
	if workspaceErr != nil {
		exists = false
		err = workspaceErr
		return
	}
	exists = true
	return
}

//DeleteWorkspace delete geoserver workspace and its reources else return error
func (g *GeoServer) DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, err error) {
	url := g.ParseURL("rest", "workspaces", workspaceName)
	httpRequest := HTTPRequest{
		Method: http.MethodDelete,
		Accept: jsonType,
		URL:    url,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
		//g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}

// GetWorkspaces get geoserver workspaces else return error
func (g *GeoServer) GetWorkspaces() (workspaces []*Resource, err error) {
	url := g.ParseURL("rest", "workspaces")
	httpRequest := HTTPRequest{
		Method: http.MethodGet,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
		//g.logger.Warn(string(response))
		workspaces = nil
		err = g.GetError(responseCode, response)
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

// GetWorkspace get geoserver workspace else return error
func (g *GeoServer) GetWorkspace(workspaceName string) (workspace Workspace, err error) {
	url := g.ParseURL("rest", "workspaces", workspaceName)
	httpRequest := HTTPRequest{
		Method: http.MethodGet,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != http.StatusOK {
		//g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}
	workspaceResponse := WorkspaceRequestBody{
		Workspace: &Workspace{},
	}
	g.DeSerializeJSON(response, &workspaceResponse)
	workspace = *workspaceResponse.Workspace
	return
}
