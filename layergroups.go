package geoserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// PublishedGroupLayers geoserver published layers
type PublishedGroupLayers []*GroupPublishableItem

// GroupPublishableItem geoserver Group
type GroupPublishableItem struct {
	Type string `json:"@type,omitempty" xml:"type"`
	Name string `json:"name,omitempty" xml:"name"`
	Href string `json:"href,omitempty" xml:"href"`
}

// LayerGroupKeywords geoserver layergroups keywords
type LayerGroupKeywords struct {
	Keyword []*string `json:"keyword,omitempty"`
}

// Publishables Geoserver Published Layers
type Publishables struct {
	Published PublishedGroupLayers `json:"published" xml:"published"`
}

// UnmarshalJSON custom deserialization to handle published layers of group.
//
// GeoServer's REST API serializes a single published layer as an object and
// multiple as an array — JSON shape that does not naturally unmarshal into a
// slice. This implementation handles both shapes safely; in v1.0.x, type
// assertions on missing fields would panic.
func (u *PublishedGroupLayers) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch raw := raw.(type) {
	case map[string]interface{}:
		name, _ := raw["name"].(string)
		href, _ := raw["href"].(string)
		typ, _ := raw["@type"].(string)
		if name == "" && href == "" && typ == "" {
			return fmt.Errorf("layergroups: unrecognized published-layer payload: %v", raw)
		}
		*u = make(PublishedGroupLayers, 0, 1)
		*u = append(*u, &GroupPublishableItem{Name: name, Href: href, Type: typ})
	case []interface{}:
		var publishedGroupLayers []*GroupPublishableItem
		if err := json.Unmarshal(data, &publishedGroupLayers); err != nil {
			return fmt.Errorf("layergroups: decode published-layer array: %w", err)
		}
		*u = publishedGroupLayers
	default:
		return fmt.Errorf("layergroups: unexpected published-layer JSON shape (%T)", raw)
	}
	return nil
}

// LayerGroupStyles geoserver layergroup styles
type LayerGroupStyles struct {
	Style []*Resource `json:"style,omitempty" xml:"style"`
}

// UnmarshalJSON tolerates GeoServer's mixed-shape `style` array. For an
// unstyled layer in the group, GeoServer emits a bare string literal (often
// "") instead of an object; for a styled layer it emits an object with
// name/href fields. Without this the standard JSON decoder errors with
// json: cannot unmarshal string into Go struct field LayerGroupStyles.style
// when the layer group has any default-styled members. String entries are
// preserved as a [Resource] with [Resource.Name] set to the string value.
func (s *LayerGroupStyles) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	// Decode into the looser intermediate type that accepts either shape.
	type rawWrapper struct {
		Style []json.RawMessage `json:"style,omitempty"`
	}
	var raw rawWrapper
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("layergroups: decode styles wrapper: %w", err)
	}

	out := make([]*Resource, 0, len(raw.Style))
	for i, elem := range raw.Style {
		if len(elem) == 0 || string(elem) == "null" {
			out = append(out, nil)
			continue
		}
		switch elem[0] {
		case '"':
			var name string
			if err := json.Unmarshal(elem, &name); err != nil {
				return fmt.Errorf("layergroups: decode style[%d] string: %w", i, err)
			}
			out = append(out, &Resource{Name: name})
		case '{':
			var r Resource
			if err := json.Unmarshal(elem, &r); err != nil {
				return fmt.Errorf("layergroups: decode style[%d] object: %w", i, err)
			}
			out = append(out, &r)
		default:
			return fmt.Errorf("layergroups: unexpected style[%d] JSON shape: %s", i, string(elem))
		}
	}
	s.Style = out
	return nil
}

// LayerGroup geoserver layergroup details
type LayerGroup struct {
	Name          string             `json:"name,omitempty" xml:"name"`
	Mode          string             `json:"mode,omitempty" xml:"mode"`
	Title         string             `json:"title,omitempty" xml:"title"`
	Workspace     *Resource          `json:"workspace,omitempty" xml:"workspace"`
	Publishables  Publishables       `json:"publishables,omitempty" xml:"publishables"`
	Styles        LayerGroupStyles   `json:"styles,omitempty" xml:"styles"`
	Bounds        NativeBoundingBox  `json:"bounds,omitempty" xml:"bounds"`
	MetadataLinks []*MetadataLink    `json:"metadataLinks,omitempty" xml:"metadataLinks"`
	Keywords      LayerGroupKeywords `json:"keywords,omitempty" xml:"keywords"`
}

type layerGroupResponse struct {
	LayerGroups struct {
		LayerGroup []*Resource `json:"layerGroup,omitempty"`
	} `json:"layerGroups,omitempty"`
}
type layerGroupDetailsResponse struct {
	LayerGroup *LayerGroup `json:"layerGroup,omitempty"`
}

// LayerGroupService define  geoserver layergroup operations
type LayerGroupService interface {
	GetLayerGroups(workspaceName string) (layerGroups []*Resource, err error)
	GetLayerGroup(workspaceName string, layerGroupName string) (layer *LayerGroup, err error)
	CreateLayerGroup(workspaceName string, layerGroup *LayerGroup) (created bool, err error)
	DeleteLayerGroup(workspaceName string, layerGroupName string) (deleted bool, err error)
}

// LayerGroupServiceWithContext is the context-aware sibling of [LayerGroupService].
type LayerGroupServiceWithContext interface {
	GetLayerGroupsContext(ctx context.Context, workspaceName string) (layerGroups []*Resource, err error)
	GetLayerGroupContext(ctx context.Context, workspaceName string, layerGroupName string) (layer *LayerGroup, err error)
	CreateLayerGroupContext(ctx context.Context, workspaceName string, layerGroup *LayerGroup) (created bool, err error)
	DeleteLayerGroupContext(ctx context.Context, workspaceName string, layerGroupName string) (deleted bool, err error)
}

// layerGroupsURL builds /rest[/workspaces/{ws}]/layergroups[/{name}].
func (g *GeoServer) layerGroupsURL(workspaceName string, extra ...string) string {
	parts := []string{"rest"}
	if workspaceName != "" {
		parts = append(parts, "workspaces", workspaceName)
	}
	parts = append(parts, "layergroups")
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// GetLayerGroups lists layer groups using context.Background.
func (g *GeoServer) GetLayerGroups(workspaceName string) (layerGroups []*Resource, err error) {
	return g.GetLayerGroupsContext(context.Background(), workspaceName)
}

// GetLayerGroupsContext is the context-aware variant of [GeoServer.GetLayerGroups].
func (g *GeoServer) GetLayerGroupsContext(ctx context.Context, workspaceName string) (layerGroups []*Resource, err error) {
	targetURL := g.layerGroupsURL(workspaceName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		layerGroups = nil
		err = g.GetError(responseCode, response)
		return
	}
	var layerGroupList layerGroupResponse
	if err = g.DeSerializeJSON(response, &layerGroupList); err != nil {
		return nil, err
	}
	layerGroups = layerGroupList.LayerGroups.LayerGroup
	return
}

// GetLayerGroup fetches a layer group using context.Background.
func (g *GeoServer) GetLayerGroup(workspaceName string, layerGroupName string) (layerGroup *LayerGroup, err error) {
	return g.GetLayerGroupContext(context.Background(), workspaceName, layerGroupName)
}

// GetLayerGroupContext is the context-aware variant of [GeoServer.GetLayerGroup].
func (g *GeoServer) GetLayerGroupContext(ctx context.Context, workspaceName string, layerGroupName string) (layerGroup *LayerGroup, err error) {
	targetURL := g.layerGroupsURL(workspaceName, layerGroupName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		layerGroup = &LayerGroup{}
		err = g.GetError(responseCode, response)
		return
	}
	var layerGroupResp layerGroupDetailsResponse
	if err = g.DeSerializeJSON(response, &layerGroupResp); err != nil {
		return nil, err
	}
	layerGroup = layerGroupResp.LayerGroup
	return
}

// CreateLayerGroup creates a layer group using context.Background.
func (g *GeoServer) CreateLayerGroup(workspaceName string, layerGroup *LayerGroup) (created bool, err error) {
	return g.CreateLayerGroupContext(context.Background(), workspaceName, layerGroup)
}

// CreateLayerGroupContext is the context-aware variant of [GeoServer.CreateLayerGroup].
func (g *GeoServer) CreateLayerGroupContext(ctx context.Context, workspaceName string, layerGroup *LayerGroup) (created bool, err error) {
	group := layerGroupDetailsResponse{LayerGroup: layerGroup}
	serializedGroup, serErr := g.SerializeStruct(group)
	if serErr != nil {
		return false, fmt.Errorf("CreateLayerGroup: serialize layer group: %w", serErr)
	}
	targetURL := g.layerGroupsURL(workspaceName)
	data := bytes.NewBuffer(serializedGroup)

	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     data,
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Error(string(response))
		created = false
		err = g.GetError(responseCode, response)
		return
	}
	created = true
	return
}

// DeleteLayerGroup deletes a layer group using context.Background.
func (g *GeoServer) DeleteLayerGroup(workspaceName string, layerGroupName string) (deleted bool, err error) {
	return g.DeleteLayerGroupContext(context.Background(), workspaceName, layerGroupName)
}

// DeleteLayerGroupContext is the context-aware variant of [GeoServer.DeleteLayerGroup].
func (g *GeoServer) DeleteLayerGroupContext(ctx context.Context, workspaceName string, layerGroupName string) (deleted bool, err error) {
	targetURL := g.layerGroupsURL(workspaceName, layerGroupName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
