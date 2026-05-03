// Package namespaces is the v2 sub-client for the GeoServer
// /rest/namespaces resource. Namespaces are GeoServer's URI-bearing
// counterpart to workspaces — every workspace has an associated
// namespace with the same name (prefix) and a configurable URI used
// in WFS / GML output.
package namespaces

import "encoding/json"

// Namespace is the GeoServer namespace document.
//
// Prefix matches the workspace name; URI is the XML-namespace URI
// used in WFS GetFeature responses for layers in this workspace.
// Isolated mirrors the workspace's isolated flag — when true,
// resources in this namespace are only addressable through their
// fully-qualified prefix:name form.
//
// Wire-format quirk: the list endpoint returns `{"name":..., "href":...}`
// for each entry while the detail endpoint returns the full
// `{"prefix":..., "uri":..., "isolated":...}` shape. The custom
// [Namespace.UnmarshalJSON] coerces both into [Namespace.Prefix].
type Namespace struct {
	Prefix   string `json:"prefix,omitempty"`
	URI      string `json:"uri,omitempty"`
	Isolated bool   `json:"isolated,omitempty"`
}

// UnmarshalJSON accepts both wire shapes — GeoServer's list endpoint
// returns `{"name":..., "href":...}` while the detail endpoint returns
// `{"prefix":..., "uri":...}`. The list-shape `name` is coerced into
// [Namespace.Prefix].
func (n *Namespace) UnmarshalJSON(data []byte) error {
	type alias Namespace
	aux := struct {
		*alias
		Name string `json:"name,omitempty"`
	}{alias: (*alias)(n)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if n.Prefix == "" && aux.Name != "" {
		n.Prefix = aux.Name
	}
	return nil
}

// Patch is a partial-update payload for [Client.Update]. Pointer
// fields let callers distinguish "field absent" from "field set to
// false / empty string".
type Patch struct {
	URI      *string `json:"uri,omitempty"`
	Isolated *bool   `json:"isolated,omitempty"`
}

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// listResponse mirrors GeoServer's `{"namespaces":{"namespace":[…]}}`.
type listResponse struct {
	Namespaces struct {
		Namespace []Namespace `json:"namespace"`
	} `json:"namespaces"`
}

// detailResponse mirrors GeoServer's `{"namespace":{…}}`.
type detailResponse struct {
	Namespace Namespace `json:"namespace"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	Namespace Namespace `json:"namespace"`
}
