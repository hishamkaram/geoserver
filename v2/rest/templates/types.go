// Package templates is the v2 sub-client for the GeoServer
// /rest/.../templates endpoint family. Templates are FreeMarker
// (FTL) files that customize GetFeatureInfo HTML output, WMS
// HTML capabilities, and other text outputs.
//
// GeoServer scopes templates at six nested levels — global, per
// workspace, per datastore, per feature type, per coverage store,
// and per coverage. Lookup walks from the most specific applicable
// scope outward to global. The SDK exposes the chain via fluent
// scoping starting from [Client]:
//
//	c.Templates                                                              // global
//	c.Templates.InWorkspace(ws)                                              // workspace
//	c.Templates.InWorkspace(ws).InDatastore(ds)                              // datastore
//	c.Templates.InWorkspace(ws).InDatastore(ds).InFeatureType(ft)            // feature type
//	c.Templates.InWorkspace(ws).InCoverageStore(cs)                          // coverage store
//	c.Templates.InWorkspace(ws).InCoverageStore(cs).InCoverage(cov)          // coverage
//
// Each scope exposes the same four methods: [Client.List],
// [Client.Get], [Client.Put], [Client.Delete].
//
// Template names: GeoServer stores templates with a ".ftl"
// extension. The SDK accepts names with or without the suffix and
// normalizes — `c.Templates.Get(ctx, "foo")` and
// `c.Templates.Get(ctx, "foo.ftl")` both target the same resource.
package templates

import "strings"

// TemplateRef is one entry in the templates listing for a scope.
type TemplateRef struct {
	// Name is the template's filename, including the ".ftl" suffix
	// (e.g. "content.ftl", "header.ftl"). Pass either form
	// (with or without ".ftl") back to [Client.Get],
	// [Client.Put], or [Client.Delete] — the SDK normalizes.
	Name string
	// Href is the absolute URL GeoServer reports for the template
	// (informational; the SDK builds its own URLs internally).
	Href string
}

// templatesWire is the GeoServer JSON envelope for a templates
// listing. Class-name wrapper keys are GeoServer's canonical wire
// shape for non-catalog list endpoints (same family as GWC's
// `org.geowebcache.diskquota.DiskQuotaConfig` etc.):
//
//	{"org.geoserver.rest.catalog.TemplateInfos":{
//	   "org.geoserver.rest.catalog.TemplateInfo":[
//	     {"name":"foo.ftl","href":"http://srv/.../foo.ftl.json"},
//	     ...
//	   ]
//	 }}
//
// Empty list collapses to an empty object on the wire shape.
type templatesWire struct {
	Infos struct {
		Info []TemplateRef `json:"org.geoserver.rest.catalog.TemplateInfo"`
	} `json:"org.geoserver.rest.catalog.TemplateInfos"`
}

// ensureFTL appends ".ftl" if the user-supplied name doesn't already
// have it. Empty input is returned unchanged so the caller sees a
// dedicated "empty name" error rather than a surprise URL.
func ensureFTL(name string) string {
	if name == "" || strings.HasSuffix(name, ".ftl") {
		return name
	}
	return name + ".ftl"
}
