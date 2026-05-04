// Package wire holds wire-format types shared across GeoServer v2
// resource sub-packages. External code must not import this package
// directly — use the exported aliases on each rest/<resource>/ package
// instead (e.g., featuretypes.CRS, coverages.CRS — both alias the same
// underlying type from this package).
//
// This package exists to eliminate copy-paste of the GIS-wide types
// (CRS, BoundingBox, Keywords, etc.) between featuretypes and
// coverages. Going through type aliases means the underlying type
// identity is shared, so values can be passed between sub-packages
// without conversion.
package wire

import (
	"encoding/json"
	"fmt"
)

// CRS models the GeoServer coordinate-reference-system field, which
// on the wire is either a JSON object ({"@class":"projected","$":"EPSG:4326"})
// or a bare string identifier. The custom Marshal / Unmarshal preserves
// this asymmetry so reads round-trip correctly.
//
// When constructing for write, set Class="" and Value="EPSG:xxxx" for
// the bare-string form; set Class="projected" / "geographic" / "string"
// (and Value to the SRS) for the object form.
type CRS struct {
	Class string `json:"@class,omitempty"`
	Value string `json:"$,omitempty"`
}

// UnmarshalJSON accepts both shapes returned by GeoServer:
//
//   - object: {"@class":"projected","$":"EPSG:4326"}
//   - bare string: "EPSG:4326"
func (c *CRS) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case map[string]any:
		class, _ := v["@class"].(string)
		value, _ := v["$"].(string)
		if class == "" && value == "" {
			return fmt.Errorf("wire: unrecognized CRS payload: %v", v)
		}
		*c = CRS{Class: class, Value: value}
	case string:
		*c = CRS{Class: "string", Value: v}
	default:
		return fmt.Errorf("wire: unrecognized CRS payload type: %T", v)
	}
	return nil
}

// MarshalJSON emits either a bare string (when Class=="string") or the
// object form. An empty CRS marshals to an empty string to match the
// wire shape GeoServer accepts on write.
func (c *CRS) MarshalJSON() ([]byte, error) {
	if c == nil || (c.Class == "" && c.Value == "") {
		return json.Marshal("")
	}
	if c.Class == "string" {
		return json.Marshal(c.Value)
	}
	return json.Marshal(struct {
		Class string `json:"@class,omitempty"`
		Value string `json:"$,omitempty"`
	}{Class: c.Class, Value: c.Value})
}

// BoundingBox is the geographic extent shared by [NativeBoundingBox]
// and [LatLonBoundingBox].
type BoundingBox struct {
	MinX float64 `json:"minx"`
	MaxX float64 `json:"maxx"`
	MinY float64 `json:"miny"`
	MaxY float64 `json:"maxy"`
}

// NativeBoundingBox is the resource extent expressed in the native
// coordinate reference system.
type NativeBoundingBox struct {
	BoundingBox
	CRS *CRS `json:"crs,omitempty"`
}

// LatLonBoundingBox is the resource extent expressed in WGS84 lat/lon.
type LatLonBoundingBox struct {
	BoundingBox
	CRS *CRS `json:"crs,omitempty"`
}

// Keywords is the keywords block on a feature type or coverage
// document.
type Keywords struct {
	String []string `json:"string,omitempty"`
}
