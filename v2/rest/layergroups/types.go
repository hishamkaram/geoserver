// Package layergroups is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/layergroups resource. A layer group bundles
// multiple layers (and optionally other layer groups) under a single
// addressable name with shared bounding box, attribution, and
// per-member style.
package layergroups

import (
	"encoding/json"
	"fmt"
)

// LayerGroup is the GeoServer layer-group document.
type LayerGroup struct {
	Name          string         `json:"name,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	Title         string         `json:"title,omitempty"`
	Workspace     *Ref           `json:"workspace,omitempty"`
	Publishables  Publishables   `json:"publishables,omitempty"`
	Styles        Styles         `json:"styles,omitempty"`
	Bounds        *Bounds        `json:"bounds,omitempty"`
	MetadataLinks []MetadataLink `json:"metadataLinks,omitempty"`
	Keywords      *Keywords      `json:"keywords,omitempty"`
}

// Ref is a generic reference object (name + href). Only Name is
// meaningful for SDK callers.
type Ref struct {
	Class string `json:"@class,omitempty"`
	Name  string `json:"name,omitempty"`
	Href  string `json:"href,omitempty"`
}

// Publishables is the wrapper around the published-layers list. The
// embedded [Published] type carries a custom Unmarshal that handles
// GeoServer's quirk of emitting one published layer as a bare object
// and multiple as an array — see [Published].
type Publishables struct {
	Published Published `json:"published,omitempty"`
}

// Published is the list of layers (and optional nested layer groups)
// that make up the group. GeoServer's REST API serializes a single
// member as a JSON object and multiple members as an array; this
// custom Unmarshal handles both shapes.
type Published []PublishedItem

// PublishedItem is one published member of the group — either a layer
// (Type="layer") or a nested layer group (Type="layerGroup"). The
// Type field carries the GeoServer wire `@type` discriminator.
type PublishedItem struct {
	Type string `json:"@type,omitempty"`
	Name string `json:"name,omitempty"`
	Href string `json:"href,omitempty"`
}

// UnmarshalJSON accepts either a single object or a JSON array. v1
// layer groups with a single member emit the object form; multi-member
// groups emit the array form.
func (p *Published) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	switch data[0] {
	case '{':
		var single PublishedItem
		if err := json.Unmarshal(data, &single); err != nil {
			return fmt.Errorf("layergroups: decode single published: %w", err)
		}
		if single.Name == "" && single.Href == "" && single.Type == "" {
			return fmt.Errorf("layergroups: empty published payload: %s", data)
		}
		*p = Published{single}
	case '[':
		type plain []PublishedItem
		var arr plain
		if err := json.Unmarshal(data, &arr); err != nil {
			return fmt.Errorf("layergroups: decode published array: %w", err)
		}
		*p = Published(arr)
	default:
		return fmt.Errorf("layergroups: unexpected published JSON shape: %s", data)
	}
	return nil
}

// Styles wraps the per-member style list. GeoServer's wire format uses
// a mixed array — string for "use the layer's default style" and
// object {name, href} for an explicit style — which Go's standard JSON
// decoder cannot handle directly. The custom Unmarshal preserves both
// shapes; string entries become a [Ref] with [Ref.Name] holding the
// raw string (often "" for "default").
type Styles struct {
	Style []Ref `json:"style,omitempty"`
}

// UnmarshalJSON tolerates GeoServer's mixed-shape `style` array.
func (s *Styles) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	type rawWrapper struct {
		Style []json.RawMessage `json:"style,omitempty"`
	}
	var raw rawWrapper
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("layergroups: decode styles wrapper: %w", err)
	}
	out := make([]Ref, 0, len(raw.Style))
	for i, elem := range raw.Style {
		if len(elem) == 0 || string(elem) == "null" {
			out = append(out, Ref{})
			continue
		}
		switch elem[0] {
		case '"':
			var name string
			if err := json.Unmarshal(elem, &name); err != nil {
				return fmt.Errorf("layergroups: decode style[%d] string: %w", i, err)
			}
			out = append(out, Ref{Name: name})
		case '{':
			var r Ref
			if err := json.Unmarshal(elem, &r); err != nil {
				return fmt.Errorf("layergroups: decode style[%d] object: %w", i, err)
			}
			out = append(out, r)
		default:
			return fmt.Errorf("layergroups: unexpected style[%d] JSON shape: %s", i, string(elem))
		}
	}
	s.Style = out
	return nil
}

// Bounds is the layer-group geographic extent.
type Bounds struct {
	MinX float64 `json:"minx"`
	MaxX float64 `json:"maxx"`
	MinY float64 `json:"miny"`
	MaxY float64 `json:"maxy"`
	CRS  string  `json:"crs,omitempty"`
}

// MetadataLink is one external metadata URL attached to the group.
type MetadataLink struct {
	Type         string `json:"type,omitempty"`
	MetadataType string `json:"metadataType,omitempty"`
	Content      string `json:"content,omitempty"`
}

// Keywords is the keyword list attached to the group.
type Keywords struct {
	Keyword []string `json:"keyword,omitempty"`
}

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// listResponse mirrors GeoServer's `{"layerGroups":{"layerGroup":[…]}}`.
type listResponse struct {
	LayerGroups struct {
		LayerGroup []LayerGroup `json:"layerGroup"`
	} `json:"layerGroups"`
}

// detailResponse mirrors GeoServer's `{"layerGroup":{…}}`.
type detailResponse struct {
	LayerGroup LayerGroup `json:"layerGroup"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	LayerGroup LayerGroup `json:"layerGroup"`
}
