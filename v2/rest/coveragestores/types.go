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

// UploadMethod selects the file-upload sub-resource on
// [WorkspaceClient.UploadFile] / [WorkspaceClient.HarvestGranule].
type UploadMethod string

// Upload methods. The default ([UploadMethodFile]) ships the file
// body across the wire; the other two reference data already
// reachable from the server.
const (
	// UploadMethodFile uploads the raster bytes in the request body.
	// Routes to `PUT /file[.<ext>]`. Default.
	UploadMethodFile UploadMethod = "file"
	// UploadMethodURL provides a URL string the server fetches.
	// Routes to `PUT /url[.<ext>]`.
	UploadMethodURL UploadMethod = "url"
	// UploadMethodExternal provides a server-local filesystem path
	// string. No file transfer happens. Routes to `PUT /external[.<ext>]`.
	UploadMethodExternal UploadMethod = "external"
)

// UploadOptions controls a [WorkspaceClient.UploadFile] or
// [WorkspaceClient.HarvestGranule] call.
type UploadOptions struct {
	// Extension selects the URL suffix (e.g., "geotiff",
	// "worldimage", "imagemosaic"). May be empty if GeoServer can
	// infer from the body, but the documented forms always include
	// an extension.
	Extension string

	// Method selects the sub-resource. Default [UploadMethodFile].
	Method UploadMethod

	// ContentType overrides the Content-Type header. Defaults are:
	//   - file:     application/zip
	//   - url:      text/plain
	//   - external: text/plain
	// Override when uploading a non-zipped binary (e.g.,
	// `image/tiff` for a single GeoTIFF granule).
	ContentType string

	// Update sets the optional `update` query parameter (typically
	// `overwrite` or `append`). Empty means use the server default.
	Update string
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
