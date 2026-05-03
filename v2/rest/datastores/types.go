// Package datastores is the v2 sub-client for the GeoServer
// /rest/workspaces/{ws}/datastores resource. The client is hierarchical:
// list/get/create/update/delete operate against a workspace-scoped view
// returned by [Client.InWorkspace].
package datastores

import "strconv"

// Datastore is the GeoServer datastore document. The same shape is used
// for list items (where only Name is populated by the server) and for
// detail responses (where every field may be populated).
//
// Workspace, Default, and Type are response-only on read paths. They
// are ignored by Create — the workspace name is taken from the URL
// scope, and the type is derived from the connection parameters.
type Datastore struct {
	Name                 string               `json:"name,omitempty"`
	Type                 string               `json:"type,omitempty"`
	Enabled              bool                 `json:"enabled,omitempty"`
	Default              bool                 `json:"_default,omitempty"`
	Workspace            *WorkspaceRef        `json:"workspace,omitempty"`
	FeatureTypes         string               `json:"featureTypes,omitempty"`
	ConnectionParameters ConnectionParameters `json:"connectionParameters,omitempty"`
}

// WorkspaceRef is the workspace pointer carried back on a Datastore
// response. Only Name is meaningful for SDK callers; the SDK builds
// URLs itself rather than following the response Href.
type WorkspaceRef struct {
	Name string `json:"name,omitempty"`
}

// ConnectionParameters is GeoServer's wire shape for datastore connection
// parameters. The on-the-wire form is `{"entry":[{"@key":"host","$":"…"}, …]}`,
// which is the XML-as-JSON convention GeoServer's REST API requires.
type ConnectionParameters struct {
	Entry []ConnectionEntry `json:"entry,omitempty"`
}

// ConnectionEntry is one key/value pair inside [ConnectionParameters].
// The JSON tags `@key` and `$` are mandatory — GeoServer rejects bodies
// using plain "key"/"value" tags.
type ConnectionEntry struct {
	Key   string `json:"@key"`
	Value string `json:"$"`
}

// Patch is a partial-update payload for [WorkspaceClient.Update].
// Pointer fields let callers distinguish "field absent" from "field
// set to false / empty string". GeoServer treats PUT as a merge-patch.
//
// Note on ConnectionParameters: GeoServer replaces the entire
// `connectionParameters` block on PUT — it does not merge entries. To
// change a single parameter, fetch the full document with [WorkspaceClient.Get],
// mutate the entries you need, and PUT the whole block back.
type Patch struct {
	Enabled              *bool                 `json:"enabled,omitempty"`
	ConnectionParameters *ConnectionParameters `json:"connectionParameters,omitempty"`
}

// ListOptions controls listing behavior. Currently empty; GeoServer's
// /rest/workspaces/{ws}/datastores does not paginate. Reserved for
// future fields and kept on the public API so adding one is non-breaking.
type ListOptions struct{}

// DeleteOptions controls Delete behavior.
type DeleteOptions struct {
	// Recurse deletes the datastore and all contained feature types
	// and layers. Default false (a non-empty datastore is rejected
	// without Recurse).
	Recurse bool
}

// Connector produces the [Datastore] payload sent to GeoServer when
// creating a datastore. The standard convenience types [PostGIS] and
// [JNDI] satisfy this interface; callers needing a different driver
// (e.g., Shapefile, GeoPackage, Oracle) can supply a [Datastore]
// directly via [Raw].
type Connector interface {
	Datastore() Datastore
}

// PostGIS describes a direct PostGIS connection. Implements [Connector].
//
// If Schema is empty, "public" is used (matching v1 default behavior
// and the GeoServer UI default). Use Extra for any additional connection
// parameters such as "max connections", "Expose primary keys",
// "preparedStatements", etc.
type PostGIS struct {
	Name     string
	Host     string
	Port     int
	Database string
	Schema   string
	User     string
	Password string
	Extra    []ConnectionEntry
}

// Datastore returns the wire-format payload for the connection.
func (p PostGIS) Datastore() Datastore {
	schema := p.Schema
	if schema == "" {
		schema = "public"
	}
	entries := make([]ConnectionEntry, 0, 7+len(p.Extra))
	entries = append(entries,
		ConnectionEntry{Key: "host", Value: p.Host},
		ConnectionEntry{Key: "port", Value: strconv.Itoa(p.Port)},
		ConnectionEntry{Key: "database", Value: p.Database},
		ConnectionEntry{Key: "schema", Value: schema},
		ConnectionEntry{Key: "user", Value: p.User},
		ConnectionEntry{Key: "passwd", Value: p.Password},
		ConnectionEntry{Key: "dbtype", Value: "postgis"},
	)
	entries = append(entries, p.Extra...)
	return Datastore{
		Name:                 p.Name,
		ConnectionParameters: ConnectionParameters{Entry: entries},
	}
}

// JNDI describes a datastore that uses a JNDI connection pool managed
// by the servlet container (typically Tomcat). Implements [Connector].
// See https://docs.geoserver.org/stable/en/user/tutorials/tomcat-jndi/tomcat-jndi.html.
type JNDI struct {
	Name              string
	DBType            string // e.g., "postgis"
	JNDIReferenceName string // e.g., "java:comp/env/jdbc/postgres"
	Extra             []ConnectionEntry
}

// Datastore returns the wire-format payload for the JNDI connection.
func (j JNDI) Datastore() Datastore {
	entries := make([]ConnectionEntry, 0, 2+len(j.Extra))
	entries = append(entries,
		ConnectionEntry{Key: "jndiReferenceName", Value: j.JNDIReferenceName},
		ConnectionEntry{Key: "dbtype", Value: j.DBType},
	)
	entries = append(entries, j.Extra...)
	return Datastore{
		Name:                 j.Name,
		ConnectionParameters: ConnectionParameters{Entry: entries},
	}
}

// Raw adapts a fully-formed [Datastore] to the [Connector] interface.
// Use it for drivers without a dedicated convenience type:
//
//	c.Datastores.InWorkspace("topp").Create(ctx, datastores.Raw(datastores.Datastore{
//	    Name: "states_shp",
//	    ConnectionParameters: datastores.ConnectionParameters{Entry: []datastores.ConnectionEntry{
//	        {Key: "url", Value: "file:data/shapefiles/states.shp"},
//	    }},
//	}))
func Raw(d Datastore) Connector { return rawConnector{d: d} }

type rawConnector struct{ d Datastore }

// Datastore returns the wrapped Datastore as-is.
func (r rawConnector) Datastore() Datastore { return r.d }

// listResponse mirrors GeoServer's `{"dataStores":{"dataStore":[…]}}` shape.
type listResponse struct {
	DataStores struct {
		DataStore []Datastore `json:"dataStore"`
	} `json:"dataStores"`
}

// detailResponse mirrors GeoServer's `{"dataStore":{…}}` shape.
type detailResponse struct {
	DataStore Datastore `json:"dataStore"`
}

// createRequest mirrors GeoServer's create body shape.
type createRequest struct {
	DataStore Datastore `json:"dataStore"`
}
