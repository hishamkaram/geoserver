// Package resources is the v2 sub-client for the GeoServer Resource
// REST API at /rest/resource/{path}. The Resource API provides
// generic byte-stream access to files in the GeoServer data
// directory — anything that is data, not configuration. Common
// uses:
//
//   - Read or write FreeMarker (FTL) templates that customize
//     GetFeatureInfo / WMS HTML output.
//   - Upload icons or external graphic files referenced from SLD
//     styles (PNG, SVG).
//   - Inspect the contents of arbitrary directories under the data
//     dir (e.g., logs/, styles/, templates/, workspaces/).
//
// The API uses a single endpoint with verb- and query-driven
// behavior: GET reads (with operation=default for content,
// operation=metadata for a metadata summary); HEAD returns just
// metadata in headers; PUT writes (with operation=default for
// upload, operation=move/copy for relocation); DELETE removes
// (recursively for directories). POST is invalid (405).
//
// This package exposes the daily-driver methods: [Client.Get],
// [Client.List], [Client.Stat], [Client.Exists], [Client.Put],
// [Client.Move], [Client.Copy], [Client.Delete].
package resources

import (
	"encoding/json"
	"errors"
	"strings"
)

// Type classifies a resource as a regular file (sometimes called a
// "leaf resource") or a directory. The wire form is the lowercase
// strings GeoServer returns in the Resource-Type response header
// and the JSON ResourceMetadata.type field.
type Type string

// Recognized resource types.
const (
	// TypeResource is a regular file resource.
	TypeResource Type = "resource"
	// TypeDirectory is a directory of child resources.
	TypeDirectory Type = "directory"
	// TypeUndefined is GeoServer's wire-quirk response for a
	// non-existent path queried via operation=metadata (instead of
	// returning 404). The [Client.Stat] method translates this into
	// an [ErrNotFound]-bearing error so callers don't have to
	// special-case it; you should not see this constant in normal
	// SDK use unless you bypass [Client.Stat] and call the wire
	// endpoint directly.
	TypeUndefined Type = "undefined"
)

// Metadata describes a single resource. For the bare metadata of a
// directory (without its children), use [Client.Stat]. For the
// full directory listing, use [Client.List].
type Metadata struct {
	// Name is the leaf name of the resource (e.g. "default_point.sld",
	// "templates"), without the parent path.
	Name string

	// ParentPath is the path of the parent directory, slash-separated,
	// e.g. "" for the root, "/" for a top-level resource, or
	// "/styles" for a file inside the styles directory. The exact
	// shape ("/" vs "" for the root) is preserved as GeoServer
	// returned it.
	ParentPath string

	// LastModified is the timestamp GeoServer reports for the
	// resource, in its native string form (e.g.
	// "2025-10-13 05:03:12.0 UTC"). Left as a string because
	// GeoServer's format is not RFC 3339; callers that need a
	// time.Time can parse with the same layout.
	LastModified string

	// Type is "resource" (regular file) or "directory".
	Type Type
}

// Directory is a directory listing — [Metadata] for the directory
// itself plus the immediate children. Children are populated only
// when the listing was retrieved via [Client.List]; [Client.Stat]
// returns just the metadata even for directories (per the upstream
// REST API's operation=metadata mode).
type Directory struct {
	Metadata
	Children []Child
}

// Child is one entry in a [Directory] listing.
type Child struct {
	// Name is the leaf name (e.g. "default_point.sld").
	Name string
	// Href is the absolute URL GeoServer reports for the child
	// resource. Useful for streaming download but not required —
	// callers can also reconstruct the path and call
	// [Client.Get] directly.
	Href string
	// MimeType is the Content-Type GeoServer guessed from the
	// child's name / contents (e.g. "text/xml", "image/png",
	// "application/octet-stream").
	MimeType string
}

// resourceDirectoryWire is the GeoServer JSON envelope for a
// directory listing — a top-level "ResourceDirectory" object.
type resourceDirectoryWire struct {
	ResourceDirectory struct {
		Name         string    `json:"name"`
		Parent       parentRef `json:"parent"`
		LastModified string    `json:"lastModified"`
		Children     struct {
			Child childArray `json:"child"`
		} `json:"children"`
	} `json:"ResourceDirectory"`
}

// resourceMetadataWire is the GeoServer JSON envelope for a
// per-resource metadata document — top-level "ResourceMetadata".
type resourceMetadataWire struct {
	ResourceMetadata struct {
		Name         string    `json:"name"`
		Parent       parentRef `json:"parent"`
		LastModified string    `json:"lastModified"`
		Type         Type      `json:"type"`
	} `json:"ResourceMetadata"`
}

// parentRef captures the parent.path field. The full envelope also
// includes a "link" sub-object with href / rel / type, which the
// SDK ignores — the parent path is the only piece callers need.
type parentRef struct {
	Path string `json:"path"`
}

// childArray is GeoServer's classic "may be a single object or an
// array of objects" wire shape. A directory with no children may
// also omit the field entirely or send it as an empty string.
type childArray []childWire

// UnmarshalJSON tolerates the three documented shapes for the
// "child" field:
//
//   - a JSON array of child objects (the canonical shape)
//   - a single child object (single-element collapse)
//   - an empty string "" (no children, GeoServer's empty-collection
//     wire form — same pattern used elsewhere in the API)
func (a *childArray) UnmarshalJSON(b []byte) error {
	trimmed := bytesTrimSpaceASCII(b)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*a = nil
		return nil
	}
	switch trimmed[0] {
	case '[':
		var arr []childWire
		if err := json.Unmarshal(b, &arr); err != nil {
			return err
		}
		*a = arr
		return nil
	case '{':
		var single childWire
		if err := json.Unmarshal(b, &single); err != nil {
			return err
		}
		*a = []childWire{single}
		return nil
	case '"':
		// Empty-collection wire form ("") — treat as no children.
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s != "" {
			return errors.New("resources: unexpected non-empty string for child")
		}
		*a = nil
		return nil
	default:
		return errors.New("resources: unexpected JSON shape for child")
	}
}

// childWire is one entry of the "child" array on the wire.
type childWire struct {
	Name string `json:"name"`
	Link struct {
		Href string `json:"href"`
		Type string `json:"type"`
	} `json:"link"`
}

// bytesTrimSpaceASCII trims leading ASCII whitespace from a byte
// slice. The encoding/json package uses the same definition.
func bytesTrimSpaceASCII(b []byte) []byte {
	for len(b) > 0 {
		c := b[0]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		b = b[1:]
	}
	return b
}

// splitPath converts a slash-separated resource path (e.g.
// "styles/foo.sld" or "/templates/getfeatureinfo.ftl") into the
// list of segments expected by [Core.URL]. Leading and trailing
// slashes are tolerated; empty segments (from "//") are dropped.
func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
