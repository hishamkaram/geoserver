// Package coveragestores is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/coveragestores resource. Coverage stores are
// the raster-side analogue of datastores: a store points at a raster
// source (GeoTIFF file, ImageMosaic directory, ArcSDE coverage, etc.)
// and individual coverages live inside the store.
package coveragestores

// CoverageStore is the GeoServer coverage-store document. The same
// shape is used for read and write paths.
//
// Workspace, Default, and Coverages are response-only on read paths:
// a Create payload should leave them zero-valued — the workspace is
// taken from the URL scope, and the rest are server-managed.
type CoverageStore struct {
	Name        string        `json:"name,omitempty"`
	URL         string        `json:"url,omitempty"`
	Description string        `json:"description,omitempty"`
	Type        string        `json:"type,omitempty"`
	Enabled     bool          `json:"enabled,omitempty"`
	Workspace   *WorkspaceRef `json:"workspace,omitempty"`
	Default     bool          `json:"_default,omitempty"`
	Coverages   string        `json:"coverages,omitempty"`
}

// WorkspaceRef is the workspace pointer carried back on a CoverageStore
// response. Only Name is meaningful for SDK callers — the SDK builds
// URLs itself rather than following the response Href.
type WorkspaceRef struct {
	Name string `json:"name,omitempty"`
}

// Patch is a partial-update payload. Pointer fields let callers
// distinguish "field absent" from "field set to false / empty string"
// when issuing an Update.
type Patch struct {
	URL         *string `json:"url,omitempty"`
	Description *string `json:"description,omitempty"`
	Type        *string `json:"type,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// ListOptions controls listing behavior. Currently empty; the
// underlying endpoint does not paginate. Reserved for future fields.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Recurse deletes the coverage store and all coverages and layers
	// configured against it. Default false rejects deletion when the
	// store contains configured coverages.
	Recurse bool
}

// listResponse mirrors GeoServer's `{"coverageStores":{"coverageStore":[…]}}`.
type listResponse struct {
	CoverageStores struct {
		CoverageStore []CoverageStore `json:"coverageStore"`
	} `json:"coverageStores"`
}

// detailResponse mirrors GeoServer's `{"coverageStore":{…}}`.
type detailResponse struct {
	CoverageStore CoverageStore `json:"coverageStore"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	CoverageStore CoverageStore `json:"coverageStore"`
}
