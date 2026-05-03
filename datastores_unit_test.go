package geoserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatastores_DatastoreConnection_GetDatastoreObj_BasicEntries(t *testing.T) {
	conn := DatastoreConnection{
		Name: "ds", Host: "localhost", Port: 5432, DBName: "gis",
		DBSchema: "public", DBUser: "u", DBPass: "p", Type: "postgis",
	}
	got := conn.GetDatastoreObj()
	assert.Equal(t, "ds", got.Name)
	keys := map[string]string{}
	for _, e := range got.ConnectionParameters.Entry {
		keys[e.Key] = e.Value
	}
	assert.Equal(t, "localhost", keys["host"])
	assert.Equal(t, "5432", keys["port"])
	assert.Equal(t, "gis", keys["database"])
	assert.Equal(t, "public", keys["schema"])
	assert.Equal(t, "postgis", keys["dbtype"])
}

func TestDatastores_DatastoreConnection_GetDatastoreObj_AppendsOptions(t *testing.T) {
	conn := DatastoreConnection{
		Name: "ds", Type: "postgis", Host: "h", Port: 1, DBName: "d", DBSchema: "s", DBUser: "u", DBPass: "p",
		Options: []Entry{
			{Key: "max connections", Value: "20"},
			{Key: "Expose primary keys", Value: "true"},
		},
	}
	got := conn.GetDatastoreObj()
	keys := map[string]string{}
	for _, e := range got.ConnectionParameters.Entry {
		keys[e.Key] = e.Value
	}
	assert.Equal(t, "20", keys["max connections"])
	assert.Equal(t, "true", keys["Expose primary keys"])
}

func TestDatastores_JNDI_GetDatastoreObj(t *testing.T) {
	conn := DatastoreJNDIConnection{
		Name:              "ds-jndi",
		Type:              "postgis",
		JndiReferenceName: "java:comp/env/jdbc/postgres",
		Options:           []Entry{{Key: "Expose primary keys", Value: "true"}},
	}
	got := conn.GetDatastoreObj()
	assert.Equal(t, "ds-jndi", got.Name)
	keys := map[string]string{}
	for _, e := range got.ConnectionParameters.Entry {
		keys[e.Key] = e.Value
	}
	assert.Equal(t, "java:comp/env/jdbc/postgres", keys["jndiReferenceName"])
	assert.Equal(t, "postgis", keys["dbtype"])
	assert.Equal(t, "true", keys["Expose primary keys"])
	// JNDI must NOT carry host/port/passwd/etc.
	for _, k := range []string{"host", "port", "passwd", "user"} {
		_, present := keys[k]
		assert.False(t, present, "JNDI datastore should not include %q", k)
	}
}

func TestDatastores_JNDI_SatisfiesConnector(t *testing.T) {
	var _ DatastoreConnector = DatastoreJNDIConnection{}
	// Pointer-to-DatastoreConnection satisfies the interface; the value form
	// does not because GetDatastoreObj has a pointer receiver on
	// DatastoreConnection. Documented in the interface docstring.
	var _ DatastoreConnector = (*DatastoreConnection)(nil)
}

func TestDatastores_CreateJNDIDatastore_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/workspaces/topp/datastores", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"jndiReferenceName"`)
		assert.Contains(t, string(body), `"dbtype"`)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateJNDIDatastoreContext(context.Background(), DatastoreJNDIConnection{
		Name:              "ds",
		Type:              "postgis",
		JndiReferenceName: "java:comp/env/jdbc/postgres",
	}, "topp")
	assert.NoError(t, err)
	assert.True(t, created)
}

func TestDatastores_CreateJNDIDatastore_409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "datastore exists")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateJNDIDatastoreContext(context.Background(), DatastoreJNDIConnection{Name: "ds", Type: "postgis", JndiReferenceName: "ref"}, "topp")
	assert.False(t, created)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestDatastores_CreateDatastoreFromConnector_DispatchesByImpl(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)

	// JNDI path
	created, err := gs.CreateDatastoreFromConnectorContext(context.Background(), DatastoreJNDIConnection{
		Name: "ds-jndi", Type: "postgis", JndiReferenceName: "java:comp/env/jdbc/x",
	}, "topp")
	assert.NoError(t, err)
	assert.True(t, created)
	assert.Contains(t, capturedBody, `"jndiReferenceName"`)

	// Direct path through the same entry point — pointer required.
	conn := DatastoreConnection{
		Name: "ds-direct", Host: "h", Port: 1, DBName: "d",
		DBSchema: "public", DBUser: "u", DBPass: "p", Type: "postgis",
	}
	created, err = gs.CreateDatastoreFromConnectorContext(context.Background(), &conn, "topp")
	assert.NoError(t, err)
	assert.True(t, created)
	assert.Contains(t, capturedBody, `"host"`)
	assert.Contains(t, capturedBody, `"dbtype"`)
}

func TestDatastores_CreateDatastore_AppliesDBSchemaDefault(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	// Pass empty DBSchema; CreateDatastoreContext should default it to
	// "public" (preserves v1.0 behavior).
	created, err := gs.CreateDatastoreContext(context.Background(), DatastoreConnection{
		Name: "ds", Host: "h", Port: 1, DBName: "d", DBSchema: "",
		DBUser: "u", DBPass: "p", Type: "postgis",
	}, "topp")
	assert.NoError(t, err)
	assert.True(t, created)
	assert.Contains(t, captured, `"$":"public"`)
}
