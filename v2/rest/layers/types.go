// Package layers is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/layers resource. A "layer" in GeoServer is the
// published rendition of a feature type or coverage — the catalog
// entity that exposes the raw data through OWS services (WMS, WFS,
// WCS) with style assignments.
//
// Layers are workspace-scoped in v2; the global /rest/layers
// endpoint is intentionally not exposed (iterate workspaces explicitly
// for a cross-workspace view).
//
// Layers are created as a side-effect of publishing a feature type or
// coverage — there is no direct Create method here. Use
// [featuretypes.DatastoreClient.Create] or [coverages.CoverageStoreClient.Create]
// to publish, then read or update the resulting layer through this
// client.
package layers

// Layer is the GeoServer layer document.
//
// Resource is a reference back to the underlying feature type or
// coverage; DefaultStyle is the WMS rendering style; Styles is the set
// of alternative styles GeoServer will offer through `?styles=`.
type Layer struct {
	Name         string       `json:"name,omitempty"`
	Path         string       `json:"path,omitempty"`
	Type         string       `json:"type,omitempty"`
	DefaultStyle *Ref         `json:"defaultStyle,omitempty"`
	Styles       *Styles      `json:"styles,omitempty"`
	Resource     *Ref         `json:"resource,omitempty"`
	Queryable    bool         `json:"queryable,omitempty"`
	Opaque       bool         `json:"opaque,omitempty"`
	Attribution  *Attribution `json:"attribution,omitempty"`
}

// Ref is a generic reference object (name + href) carried in layer
// responses. Only Name is meaningful for SDK callers.
type Ref struct {
	Class string `json:"@class,omitempty"`
	Name  string `json:"name,omitempty"`
	Href  string `json:"href,omitempty"`
}

// Styles wraps the list of style references attached to a layer. The
// `@class` field is a wire-format hint emitted by GeoServer to indicate
// the underlying Java collection type and is preserved on round-trip.
type Styles struct {
	Class string `json:"@class,omitempty"`
	Style []Ref  `json:"style,omitempty"`
}

// Attribution is the WMS GetCapabilities attribution block — credit
// line, logo URL, and dimensions. Most callers leave this nil.
type Attribution struct {
	Title      string `json:"title,omitempty"`
	Href       string `json:"href,omitempty"`
	LogoURL    string `json:"logoURL,omitempty"`
	LogoType   string `json:"logoType,omitempty"`
	LogoWidth  int    `json:"logoWidth,omitempty"`
	LogoHeight int    `json:"logoHeight,omitempty"`
}

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Recurse also deletes the underlying feature type or coverage
	// the layer points at. Default false leaves the data resource
	// intact and only removes the layer.
	Recurse bool
}

// AddStyleOptions controls a [WorkspaceClient.AddStyle] call.
type AddStyleOptions struct {
	// Default also promotes the added style to the layer's default
	// style atomically. Without this flag, the style is added as an
	// alternative renderer that callers can request via WMS
	// `?styles=<name>` but the layer's default style is unchanged.
	Default bool
}

// stylesListResponse mirrors GeoServer's `{"styles":{"style":[…]}}`
// returned by GET /layers/{l}/styles. Empty-collection envelope
// (`{"styles":""}`) is handled by the caller via the same idiom v2
// uses for the global styles list.
type stylesListResponse struct {
	Styles struct {
		Style []Ref `json:"style"`
	} `json:"styles"`
}

// addStyleRequest mirrors the wire body for POST /layers/{l}/styles —
// `{"style":{"name":"<style>"}}`.
type addStyleRequest struct {
	Style addStylePayload `json:"style"`
}

type addStylePayload struct {
	Name string `json:"name"`
}

// listResponse mirrors GeoServer's `{"layers":{"layer":[…]}}`.
type listResponse struct {
	Layers struct {
		Layer []Layer `json:"layer"`
	} `json:"layers"`
}

// detailResponse mirrors GeoServer's `{"layer":{…}}`.
type detailResponse struct {
	Layer Layer `json:"layer"`
}
