package geoserver

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
)

// DatastoreService define all geoserver datastore operations
type DatastoreService interface {

	// DatastoreExists checks if a datastore exists in a workspace else return error
	DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error)

	// GetDatastores return datastores in a workspace else return error
	GetDatastores(workspaceName string) (datastores []*Resource, err error)

	// GetDatastoreDetails get specific datastore from geoserver else return error
	GetDatastoreDetails(workspaceName string, datastoreName string) (datastore *Datastore, err error)

	// CreateDatastore create a datastore under provided workspace
	CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error)

	// CreateJNDIDatastore creates a datastore that uses a JNDI connection pool
	// (e.g., a Tomcat-managed JDBC pool). See [GeoServer.CreateJNDIDatastoreContext].
	CreateJNDIDatastore(connection DatastoreJNDIConnection, workspaceName string) (created bool, err error)

	// CreateDatastoreFromConnector creates a datastore from any value
	// implementing [DatastoreConnector]. Useful when callers want a single
	// code path for both direct and JNDI connections, or for custom
	// connector types.
	CreateDatastoreFromConnector(connector DatastoreConnector, workspaceName string) (created bool, err error)

	// DeleteDatastore deletes a datastore from geoserver else return error
	DeleteDatastore(workspaceName string, datastoreName string, recurse bool) (deleted bool, err error)
}

// DatastoreServiceWithContext is the context-aware sibling of [DatastoreService].
type DatastoreServiceWithContext interface {
	DatastoreExistsContext(ctx context.Context, workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error)
	GetDatastoresContext(ctx context.Context, workspaceName string) (datastores []*Resource, err error)
	GetDatastoreDetailsContext(ctx context.Context, workspaceName string, datastoreName string) (datastore *Datastore, err error)
	CreateDatastoreContext(ctx context.Context, datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error)
	CreateJNDIDatastoreContext(ctx context.Context, connection DatastoreJNDIConnection, workspaceName string) (created bool, err error)
	CreateDatastoreFromConnectorContext(ctx context.Context, connector DatastoreConnector, workspaceName string) (created bool, err error)
	DeleteDatastoreContext(ctx context.Context, workspaceName string, datastoreName string, recurse bool) (deleted bool, err error)
}

// DatastoreConnector produces the [Datastore] payload sent to GeoServer when
// creating a datastore. Both [*DatastoreConnection] and [DatastoreJNDIConnection]
// satisfy this interface.
//
// Note the receiver asymmetry: DatastoreConnection has a pointer-receiver
// GetDatastoreObj method (preserved from v1.0), so its pointer type
// (*DatastoreConnection) satisfies DatastoreConnector. DatastoreJNDIConnection
// uses a value-receiver method, so both DatastoreJNDIConnection and
// *DatastoreJNDIConnection satisfy. Pass &conn for the former, jndiConn (or &jndiConn) for the latter.
type DatastoreConnector interface {
	GetDatastoreObj() Datastore
}

// Datastore holds geoserver store information
type Datastore struct {
	Name                 string                    `json:"name,omitempty"`
	Href                 string                    `json:"href,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Enabled              bool                      `json:"enabled,omitempty"`
	Workspace            *Workspace                `json:"workspace,omitempty"`
	Default              bool                      `json:"_default,omitempty"`
	FeatureTypes         string                    `json:"featureTypes,omitempty"`
	ConnectionParameters DatastoreConnectionParams `json:"connectionParameters,omitempty"`
}

// DatastoreDetails this struct to send and accept json data from/to geoserver
type DatastoreDetails struct {
	Datastore *Datastore `json:"dataStore"`
}

// DatastoreConnection holds parameters to create new datastore in geoserver.
//
// The Options field (added in v1.1) carries arbitrary additional connection
// parameters such as "max connections", "Expose primary keys", etc. — see the
// GeoServer connection-parameters reference for valid keys per dbtype. New
// fields are appended to preserve positional struct-literal compatibility for
// v1.0 callers that named only the original fields.
type DatastoreConnection struct {
	Name     string
	Host     string
	Port     int
	DBName   string
	DBSchema string
	DBUser   string
	DBPass   string
	Type     string
	Options  []Entry // additional connection parameters (v1.1+)
}

// DatastoreConnectionParams in datastore json
type DatastoreConnectionParams struct {
	Entry []*Entry `json:"entry,omitempty"`
}

// DatastoreJNDIConnection holds parameters to create a datastore that uses a
// JNDI connection pool managed by the servlet container (typically Tomcat).
// See https://docs.geoserver.org/stable/en/user/tutorials/tomcat-jndi/tomcat-jndi.html
type DatastoreJNDIConnection struct {
	Name              string  // datastore name
	Type              string  // dbtype (e.g., "postgis")
	JndiReferenceName string  // e.g., "java:comp/env/jdbc/postgres"
	Options           []Entry // additional connection parameters
}

// GetDatastoreObj return datastore Object to send to geoserver rest.
//
// Options entries (v1.1+) are appended after the standard parameters; an
// empty Options slice is a no-op so the byte-for-byte output is identical
// to v1.0 for connections that don't use Options.
func (connection *DatastoreConnection) GetDatastoreObj() (datastore Datastore) {
	entries := make([]*Entry, 0, 7+len(connection.Options))
	entries = append(entries,
		&Entry{Key: "host", Value: connection.Host},
		&Entry{Key: "port", Value: strconv.Itoa(connection.Port)},
		&Entry{Key: "database", Value: connection.DBName},
		&Entry{Key: "schema", Value: connection.DBSchema},
		&Entry{Key: "user", Value: connection.DBUser},
		&Entry{Key: "passwd", Value: connection.DBPass},
		&Entry{Key: "dbtype", Value: connection.Type},
	)
	for i := range connection.Options {
		entries = append(entries, &connection.Options[i])
	}
	datastore = Datastore{
		Name:                 connection.Name,
		ConnectionParameters: DatastoreConnectionParams{Entry: entries},
	}
	return
}

// GetDatastoreObj return datastore object for a JNDI-pool-backed datastore.
func (connection DatastoreJNDIConnection) GetDatastoreObj() (datastore Datastore) {
	entries := make([]*Entry, 0, 2+len(connection.Options))
	entries = append(entries,
		&Entry{Key: "jndiReferenceName", Value: connection.JndiReferenceName},
		&Entry{Key: "dbtype", Value: connection.Type},
	)
	for i := range connection.Options {
		entries = append(entries, &connection.Options[i])
	}
	datastore = Datastore{
		Name:                 connection.Name,
		ConnectionParameters: DatastoreConnectionParams{Entry: entries},
	}
	return
}

// DatastoreExists checks if a datastore exists in a workspace using context.Background.
func (g *GeoServer) DatastoreExists(workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error) {
	return g.DatastoreExistsContext(context.Background(), workspaceName, datastoreName, quietOnNotFound)
}

// DatastoreExistsContext is the context-aware variant of [GeoServer.DatastoreExists].
func (g *GeoServer) DatastoreExistsContext(ctx context.Context, workspaceName string, datastoreName string, quietOnNotFound bool) (exists bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"quietOnNotFound": strconv.FormatBool(quietOnNotFound)},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		exists = false
		err = g.GetError(responseCode, response)
		return
	}
	exists = true
	return
}

// GetDatastores returns datastores in a workspace using context.Background.
func (g *GeoServer) GetDatastores(workspaceName string) (datastores []*Resource, err error) {
	return g.GetDatastoresContext(context.Background(), workspaceName)
}

// GetDatastoresContext is the context-aware variant of [GeoServer.GetDatastores].
func (g *GeoServer) GetDatastoresContext(ctx context.Context, workspaceName string) (datastores []*Resource, err error) {
	//TODO: check if workspace exist before creating it
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		datastores = nil
		err = g.GetError(responseCode, response)
		return
	}
	var query struct {
		DataStores struct {
			DataStore []*Resource
		}
	}
	if err = g.DeSerializeJSON(response, &query); err != nil {
		return nil, err
	}
	datastores = query.DataStores.DataStore
	return
}

// GetDatastoreDetails returns the full datastore document using context.Background.
func (g *GeoServer) GetDatastoreDetails(workspaceName string, datastoreName string) (datastore *Datastore, err error) {
	return g.GetDatastoreDetailsContext(context.Background(), workspaceName, datastoreName)
}

// GetDatastoreDetailsContext is the context-aware variant of [GeoServer.GetDatastoreDetails].
func (g *GeoServer) GetDatastoreDetailsContext(ctx context.Context, workspaceName string, datastoreName string) (datastore *Datastore, err error) {
	//TODO: check if workspace exist before creating it
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		datastore = &Datastore{}
		err = g.GetError(responseCode, response)
		return

	}
	var query DatastoreDetails
	if err = g.DeSerializeJSON(response, &query); err != nil {
		return &Datastore{}, err
	}
	datastore = query.Datastore
	return

}

// CreateDatastore creates a datastore using context.Background.
func (g *GeoServer) CreateDatastore(datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error) {
	return g.CreateDatastoreContext(context.Background(), datastoreConnection, workspaceName)
}

// CreateDatastoreContext is the context-aware variant of [GeoServer.CreateDatastore].
func (g *GeoServer) CreateDatastoreContext(ctx context.Context, datastoreConnection DatastoreConnection, workspaceName string) (created bool, err error) {
	if datastoreConnection.DBSchema == "" {
		datastoreConnection.DBSchema = "public"
	}
	return g.postDatastore(ctx, &datastoreConnection, workspaceName, "CreateDatastore")
}

// CreateJNDIDatastore creates a JNDI-pool-backed datastore using context.Background.
func (g *GeoServer) CreateJNDIDatastore(connection DatastoreJNDIConnection, workspaceName string) (created bool, err error) {
	return g.CreateJNDIDatastoreContext(context.Background(), connection, workspaceName)
}

// CreateJNDIDatastoreContext is the context-aware variant of [GeoServer.CreateJNDIDatastore].
func (g *GeoServer) CreateJNDIDatastoreContext(ctx context.Context, connection DatastoreJNDIConnection, workspaceName string) (created bool, err error) {
	return g.postDatastore(ctx, connection, workspaceName, "CreateJNDIDatastore")
}

// CreateDatastoreFromConnector creates a datastore from any [DatastoreConnector]
// using context.Background.
func (g *GeoServer) CreateDatastoreFromConnector(connector DatastoreConnector, workspaceName string) (created bool, err error) {
	return g.CreateDatastoreFromConnectorContext(context.Background(), connector, workspaceName)
}

// CreateDatastoreFromConnectorContext is the context-aware variant of
// [GeoServer.CreateDatastoreFromConnector]. The connector is responsible for
// producing a fully-formed [Datastore] payload — no defaults are applied.
func (g *GeoServer) CreateDatastoreFromConnectorContext(ctx context.Context, connector DatastoreConnector, workspaceName string) (created bool, err error) {
	return g.postDatastore(ctx, connector, workspaceName, "CreateDatastoreFromConnector")
}

// postDatastore is the shared HTTP path for the Create*Datastore* methods.
// op identifies the public entry point for error wrapping.
func (g *GeoServer) postDatastore(ctx context.Context, connector DatastoreConnector, workspaceName, op string) (created bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores")
	store := connector.GetDatastoreObj()
	data, serErr := g.SerializeStruct(DatastoreDetails{Datastore: &store})
	if serErr != nil {
		return false, fmt.Errorf("%s: serialize datastore: %w", op, serErr)
	}
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(data),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteDatastore deletes a datastore using context.Background.
func (g *GeoServer) DeleteDatastore(workspaceName string, datastoreName string, recurse bool) (deleted bool, err error) {
	return g.DeleteDatastoreContext(context.Background(), workspaceName, datastoreName, recurse)
}

// DeleteDatastoreContext is the context-aware variant of [GeoServer.DeleteDatastore].
func (g *GeoServer) DeleteDatastoreContext(ctx context.Context, workspaceName string, datastoreName string, recurse bool) (deleted bool, err error) {
	targetURL := g.ParseURL("rest", "workspaces", workspaceName, "datastores", datastoreName)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  map[string]string{"recurse": strconv.FormatBool(recurse)},
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		deleted = false
		err = g.GetError(responseCode, response)
		return
	}
	deleted = true
	return
}
