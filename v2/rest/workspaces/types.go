// Package workspaces is the v2 sub-client for the GeoServer
// /rest/workspaces resource. The exported surface is intentionally
// shaped like every other v2 resource sub-client so callers learn the
// pattern once.
package workspaces

// Workspace is the GeoServer workspace document. Fields are the subset
// callers and the GeoServer JSON wire format both use.
type Workspace struct {
	Name     string `json:"name"`
	Isolated bool   `json:"isolated,omitempty"`
}

// WorkspacePatch is a partial-update payload. Pointer fields let callers
// distinguish "field absent" from "field set to false / empty string"
// when issuing an Update — GeoServer treats PUT as a merge-patch.
type WorkspacePatch struct {
	Isolated *bool `json:"isolated,omitempty"`
}

// ListOptions controls listing behavior. Currently empty; GeoServer's
// /rest/workspaces does not paginate. Reserved for future fields and
// kept on the public API so a future paginating release is non-breaking.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Recurse deletes the workspace and all contained datastores,
	// coverage stores, layer groups, etc. Default false (non-empty
	// workspaces 403 without Recurse).
	Recurse bool
}

// listResponse mirrors GeoServer's `{"workspaces":{"workspace":[…]}}`.
type listResponse struct {
	Workspaces struct {
		Workspace []Workspace `json:"workspace"`
	} `json:"workspaces"`
}

// detailResponse mirrors GeoServer's `{"workspace":{…}}`.
type detailResponse struct {
	Workspace Workspace `json:"workspace"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	Workspace Workspace `json:"workspace"`
}
