// Package styles is the v2 sub-client for the GeoServer
// /rest/styles and /rest/workspaces/{ws}/styles resources. Styles can
// be either global (visible to every workspace) or workspace-scoped.
//
// Construct an operating client by choosing scope:
//
//	c.Styles                          // global scope (default)
//	c.Styles.InWorkspace("topp")       // workspace-scoped
//
// Styles have a JSON metadata document (name, format, filename,
// languageVersion) plus an SLD body uploaded separately via
// [Client.UploadSLD]. [Client.Create] registers the metadata only —
// follow with UploadSLD to attach the SLD content.
package styles

// Style is the GeoServer style metadata document. The SLD body itself
// is content-typed differently (application/vnd.ogc.sld+xml) and lives
// outside this struct — fetch via [Client.DownloadSLD] (deferred to a
// later PR) or upload via [Client.UploadSLD].
type Style struct {
	Name            string           `json:"name,omitempty"`
	Format          string           `json:"format,omitempty"`
	Filename        string           `json:"filename,omitempty"`
	LanguageVersion *LanguageVersion `json:"languageVersion,omitempty"`
}

// LanguageVersion identifies which SLD spec version a style targets
// (e.g., "1.0.0", "1.1.0").
type LanguageVersion struct {
	Version string `json:"version,omitempty"`
}

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Purge also removes the on-disk SLD file from the GeoServer
	// data directory. Default false leaves the file in place — only
	// the catalog reference is removed.
	Purge bool
}

// UploadOptions controls SLD body upload behavior.
type UploadOptions struct {
	// Format is the wire Content-Type for the SLD body. Defaults to
	// "application/vnd.ogc.sld+xml" (SLD 1.0 / SE 1.0). Use
	// "application/vnd.ogc.se+xml" for Symbology Encoding 1.1.0,
	// or "application/vnd.geoserver.geocss+css" for GeoCSS.
	Format string
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	Style Style `json:"style"`
}

// detailResponse mirrors GeoServer's `{"style":{…}}`.
type detailResponse struct {
	Style Style `json:"style"`
}
