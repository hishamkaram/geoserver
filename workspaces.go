package geoserver

import (
	"bytes"
	"context"
	"fmt"
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

	// DeleteWorkspace delete geoserver workspace and its reources else return error
	DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, err error)
}

// WorkspaceServiceWithContext is the context-aware sibling of [WorkspaceService].
// Every method takes a [context.Context] as first argument so callers can
// honour deadlines and cancellation. New code should prefer this interface.
type WorkspaceServiceWithContext interface {
	WorkspaceExistsContext(ctx context.Context, workspaceName string) (exists bool, err error)
	GetWorkspacesContext(ctx context.Context) (workspaces []*Resource, err error)
	GetWorkspaceContext(ctx context.Context, workspaceName string) (workspace Workspace, err error)
	CreateWorkspaceContext(ctx context.Context, workspaceName string) (created bool, err error)
	DeleteWorkspaceContext(ctx context.Context, workspaceName string, recurse bool) (deleted bool, err error)
}

// Workspace is the Workspace Object
type Workspace struct {
	Name           string `json:"name,omitempty"`
	Isolated       bool   `json:"isolated,omitempty"`
	DataStores     string `json:"dataStores,omitempty"`
	CoverageStores string `json:"coverageStores,omitempty"`
	WmsStores      string `json:"wmsStores,omitempty"`
	WmtsStores     string `json:"wmtsStores,omitempty"`
}

// WorkspaceRequestBody is the api body
type WorkspaceRequestBody struct {
	Workspace *Workspace `json:"workspace,omitempty"`
}

// CreateWorkspace creates a workspace using context.Background. See
// [GeoServer.CreateWorkspaceContext] for the cancellable variant.
func (g *GeoServer) CreateWorkspace(workspaceName string) (created bool, err error) {
	return g.CreateWorkspaceContext(context.Background(), workspaceName)
}

// CreateWorkspaceContext creates a workspace and returns whether it was
// created and any error.
func (g *GeoServer) CreateWorkspaceContext(ctx context.Context, workspaceName string) (created bool, err error) {
	workspace := Workspace{Name: workspaceName}
	serializedWorkspace, serErr := g.SerializeStruct(WorkspaceRequestBody{Workspace: &workspace})
	if serErr != nil {
		return false, fmt.Errorf("CreateWorkspace: serialize workspace: %w", serErr)
	}
	targetURL := g.ParseURL("rest", "workspaces")
	data := bytes.NewBuffer(serializedWorkspace)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     data,
		DataType: jsonType + "; charset=utf-8",
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// WorkspaceExists checks whether workspaceName exists, using context.Background.
func (g *GeoServer) WorkspaceExists(workspaceName string) (exists bool, err error) {
	return g.WorkspaceExistsContext(context.Background(), workspaceName)
}

// WorkspaceExistsContext checks whether workspaceName exists.
func (g *GeoServer) WorkspaceExistsContext(ctx context.Context, workspaceName string) (exists bool, err error) {
	_, workspaceErr := g.GetWorkspaceContext(ctx, workspaceName)
	if workspaceErr != nil {
		return false, workspaceErr
	}
	return true, nil
}

// DeleteWorkspace deletes workspaceName, using context.Background.
func (g *GeoServer) DeleteWorkspace(workspaceName string, recurse bool) (deleted bool, err error) {
	return g.DeleteWorkspaceContext(context.Background(), workspaceName, recurse)
}

// DeleteWorkspaceContext deletes workspaceName.
func (g *GeoServer) DeleteWorkspaceContext(ctx context.Context, workspaceName string, recurse bool) (deleted bool, err error) {
	url := g.ParseURL("rest", "workspaces", workspaceName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    url,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}

// GetWorkspaces lists workspaces using context.Background.
func (g *GeoServer) GetWorkspaces() (workspaces []*Resource, err error) {
	return g.GetWorkspacesContext(context.Background())
}

// GetWorkspacesContext lists workspaces.
func (g *GeoServer) GetWorkspacesContext(ctx context.Context) (workspaces []*Resource, err error) {
	url := g.ParseURL("rest", "workspaces")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		workspaces = nil
		err = g.GetError(responseCode, response)
		return
	}
	var workspaceResponse struct {
		Workspaces struct {
			Workspace []*Resource
		}
	}
	if err = g.DeSerializeJSON(response, &workspaceResponse); err != nil {
		return nil, err
	}
	workspaces = workspaceResponse.Workspaces.Workspace
	return
}

// GetWorkspace fetches a single workspace using context.Background.
func (g *GeoServer) GetWorkspace(workspaceName string) (workspace Workspace, err error) {
	return g.GetWorkspaceContext(context.Background(), workspaceName)
}

// GetWorkspaceContext fetches a single workspace.
func (g *GeoServer) GetWorkspaceContext(ctx context.Context, workspaceName string) (workspace Workspace, err error) {
	url := g.ParseURL("rest", "workspaces", workspaceName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    url,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}
	workspaceResponse := WorkspaceRequestBody{
		Workspace: &Workspace{},
	}
	if err = g.DeSerializeJSON(response, &workspaceResponse); err != nil {
		return Workspace{}, err
	}
	workspace = *workspaceResponse.Workspace
	return
}
