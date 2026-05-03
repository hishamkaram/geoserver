package datastores_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
)

// newTestClient constructs a v2 *Client pointed at the given httptest
// server, with basic-auth and a 5s timeout.
func newTestClient(t *testing.T, srv *httptest.Server) *geoserver.Client {
	t.Helper()
	c, err := geoserver.New(srv.URL,
		geoserver.WithBasicAuth("admin", "geoserver"),
		geoserver.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// expectBasicAuth asserts the Authorization header carries the
// admin/geoserver basic-auth value.
func expectBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:geoserver"))
	if got := r.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization header = %q, want %q", got, want)
	}
}

// expectUserAgent asserts the User-Agent header is the default v2 UA.
func expectUserAgent(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("User-Agent"); got != "geoserver-go/v2" {
		t.Fatalf("User-Agent = %q, want %q", got, "geoserver-go/v2")
	}
}

// findEntry returns the value of the first ConnectionEntry with the
// given key, or "" if not found.
func findEntry(entries []datastores.ConnectionEntry, key string) string {
	for _, e := range entries {
		if e.Key == key {
			return e.Value
		}
	}
	return ""
}

// ---- Connector wire-format tests (no HTTP) ----

func TestPostGIS_Datastore_DefaultSchema(t *testing.T) {
	conn := datastores.PostGIS{
		Name: "states", Host: "db", Port: 5432, Database: "gis",
		User: "u", Password: "p",
	}
	d := conn.Datastore()
	if d.Name != "states" {
		t.Fatalf("Name = %q", d.Name)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "schema"); got != "public" {
		t.Fatalf("default schema = %q, want public", got)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "dbtype"); got != "postgis" {
		t.Fatalf("dbtype = %q", got)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "port"); got != "5432" {
		t.Fatalf("port = %q", got)
	}
}

func TestPostGIS_Datastore_ExplicitSchemaAndExtra(t *testing.T) {
	conn := datastores.PostGIS{
		Name: "states", Host: "db", Port: 5432, Database: "gis",
		Schema: "myschema", User: "u", Password: "p",
		Extra: []datastores.ConnectionEntry{
			{Key: "max connections", Value: "20"},
			{Key: "Expose primary keys", Value: "true"},
		},
	}
	d := conn.Datastore()
	if got := findEntry(d.ConnectionParameters.Entry, "schema"); got != "myschema" {
		t.Fatalf("schema = %q", got)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "max connections"); got != "20" {
		t.Fatalf("max connections = %q", got)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "Expose primary keys"); got != "true" {
		t.Fatalf("Expose primary keys = %q", got)
	}
}

func TestJNDI_Datastore(t *testing.T) {
	conn := datastores.JNDI{
		Name: "states", DBType: "postgis",
		JNDIReferenceName: "java:comp/env/jdbc/postgres",
	}
	d := conn.Datastore()
	if d.Name != "states" {
		t.Fatalf("Name = %q", d.Name)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "jndiReferenceName"); got != "java:comp/env/jdbc/postgres" {
		t.Fatalf("jndiReferenceName = %q", got)
	}
	if got := findEntry(d.ConnectionParameters.Entry, "dbtype"); got != "postgis" {
		t.Fatalf("dbtype = %q", got)
	}
}

func TestRaw_Datastore(t *testing.T) {
	want := datastores.Datastore{
		Name: "states_shp",
		ConnectionParameters: datastores.ConnectionParameters{Entry: []datastores.ConnectionEntry{
			{Key: "url", Value: "file:data/shapefiles/states.shp"},
		}},
	}
	got := datastores.Raw(want).Datastore()
	if got.Name != want.Name {
		t.Fatalf("Name = %q", got.Name)
	}
	if findEntry(got.ConnectionParameters.Entry, "url") != "file:data/shapefiles/states.shp" {
		t.Fatalf("url entry not preserved: %+v", got.ConnectionParameters.Entry)
	}
}

// ---- HTTP CRUD tests ----

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		expectUserAgent(t, r)
		if r.Method != http.MethodGet || r.URL.Path != "/rest/workspaces/topp/datastores" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"dataStores":{"dataStore":[{"name":"states_shp"},{"name":"taz_shapes"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Datastores.InWorkspace("topp").List(context.Background(), datastores.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "states_shp" || got[1].Name != "taz_shapes" {
		t.Fatalf("unexpected datastores: %+v", got)
	}
}

// Regression for v1 issue #22: GeoServer 2.28+ returns `{"dataStores":""}`
// (a bare string) for an empty datastore collection rather than the
// expected `{"dataStores":{"dataStore":[]}}` shape. List must accept both.
func TestList_EmptyCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"dataStores":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Datastores.InWorkspace("empty").List(context.Background(), datastores.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for empty collection, got %+v", got)
	}
}

func TestList_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "no such workspace")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Datastores.InWorkspace("missing").List(context.Background(), datastores.ListOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	var apiErr *geoserver.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *geoserver.APIError, got %T", err)
	}
	if apiErr.Op != "Datastores.List" {
		t.Fatalf("APIError.Op = %q, want Datastores.List", apiErr.Op)
	}
}

func TestList_EmptyWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Datastores.InWorkspace("").List(context.Background(), datastores.ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "empty workspace") {
		t.Fatalf("expected empty-workspace error, got %v", err)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"dataStores":{"dataStore":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for ds, err := range c.Datastores.InWorkspace("topp").Iter(context.Background(), datastores.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, ds.Name)
	}
	if len(names) != 3 || names[0] != "a" || names[2] != "c" {
		t.Fatalf("iterator yielded %v", names)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/datastores/states_shp" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"dataStore":{"name":"states_shp","type":"Shapefile","enabled":true,"workspace":{"name":"topp"},"connectionParameters":{"entry":[{"@key":"url","$":"file:data/shapefiles/states.shp"}]}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ds, err := c.Datastores.InWorkspace("topp").Get(context.Background(), "states_shp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ds.Name != "states_shp" || ds.Type != "Shapefile" || !ds.Enabled {
		t.Fatalf("Datastore = %+v", ds)
	}
	if ds.Workspace == nil || ds.Workspace.Name != "topp" {
		t.Fatalf("Workspace = %+v", ds.Workspace)
	}
	if findEntry(ds.ConnectionParameters.Entry, "url") != "file:data/shapefiles/states.shp" {
		t.Fatalf("entries = %+v", ds.ConnectionParameters.Entry)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Datastores.InWorkspace("topp").Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_EmptyName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Datastores.InWorkspace("topp").Get(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Fatalf("expected empty-name error, got %v", err)
	}
}

func TestCreate_PostGIS_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/workspaces/topp/datastores" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		// Spot-check: must contain dbtype=postgis and the connection
		// params block. Don't assert exact byte match — JSON field
		// ordering of map-of-pointers is stable in encoding/json but
		// the entry slice order is what we care about.
		s := string(body)
		for _, sub := range []string{
			`"name":"states"`,
			`"@key":"host"`, `"$":"db"`,
			`"@key":"port"`, `"$":"5432"`,
			`"@key":"database"`, `"$":"gis"`,
			`"@key":"schema"`, `"$":"public"`,
			`"@key":"dbtype"`, `"$":"postgis"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Create(context.Background(), datastores.PostGIS{
		Name: "states", Host: "db", Port: 5432, Database: "gis",
		User: "u", Password: "p",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_JNDI_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, sub := range []string{
			`"name":"states"`,
			`"@key":"jndiReferenceName"`, `"$":"java:comp/env/jdbc/postgres"`,
			`"@key":"dbtype"`, `"$":"postgis"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Create(context.Background(), datastores.JNDI{
		Name: "states", DBType: "postgis",
		JNDIReferenceName: "java:comp/env/jdbc/postgres",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Raw_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"name":"states_shp"`) ||
			!strings.Contains(s, `"@key":"url"`) ||
			!strings.Contains(s, `"$":"file:data/shapefiles/states.shp"`) {
			t.Errorf("body = %s", s)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Create(context.Background(),
		datastores.Raw(datastores.Datastore{
			Name: "states_shp",
			ConnectionParameters: datastores.ConnectionParameters{Entry: []datastores.ConnectionEntry{
				{Key: "url", Value: "file:data/shapefiles/states.shp"},
			}},
		}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "exists")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Create(context.Background(), datastores.PostGIS{
		Name: "dup", Host: "db", Port: 5432, Database: "gis",
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_NilConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil connector") {
		t.Fatalf("expected nil-connector error, got %v", err)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Create(context.Background(), datastores.PostGIS{
		Host: "db", Port: 5432, Database: "gis",
	})
	if err == nil || !strings.Contains(err.Error(), "empty datastore Name") {
		t.Fatalf("expected empty-name error, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/workspaces/topp/datastores/states_shp" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"dataStore":{"enabled":true}}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	enabled := true
	err := c.Datastores.InWorkspace("topp").Update(context.Background(), "states_shp",
		&datastores.Patch{Enabled: &enabled})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NilPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Update(context.Background(), "states_shp", nil)
	if err == nil || !strings.Contains(err.Error(), "nil patch") {
		t.Fatalf("expected nil-patch error, got %v", err)
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/workspaces/topp/datastores/states_shp" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q, want true", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Delete(context.Background(), "states_shp",
		datastores.DeleteOptions{Recurse: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NoRecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("recurse") != "false" {
			t.Errorf("recurse = %q, want false", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Delete(context.Background(), "states_shp",
		datastores.DeleteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("topp").Delete(context.Background(), "states_shp",
		datastores.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

// URL-escaping regression: workspace and datastore names with characters
// that PathEscape encodes to "%..." sequences must produce a single-encoded
// URL on the wire (not double-encoded "%25...").
func TestGet_URLEscaping_BothSegments(t *testing.T) {
	var capturedRequestURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"dataStore":{"name":"weird"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Datastores.InWorkspace("ws*1").Get(context.Background(), "ds*name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedRequestURI, "ws%2A1") || !strings.Contains(capturedRequestURI, "ds%2Aname") {
		t.Fatalf("expected single-encoded %%2A in both segments, got %q", capturedRequestURI)
	}
	if strings.Contains(capturedRequestURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedRequestURI)
	}
}

func TestUploadFile_Default(t *testing.T) {
	var captured struct {
		Method, Path, ContentType, Accept string
		Body                              []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		captured.Accept = r.Header.Get("Accept")
		captured.Body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body := strings.NewReader("FAKEZIP")
	if err := c.Datastores.InWorkspace("topp").UploadFile(context.Background(), "states_shp", body,
		datastores.UploadOptions{Extension: "shp"}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if captured.Method != http.MethodPut {
		t.Errorf("Method = %q, want PUT", captured.Method)
	}
	if captured.Path != "/rest/workspaces/topp/datastores/states_shp/file.shp" {
		t.Errorf("Path = %q", captured.Path)
	}
	if captured.ContentType != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", captured.ContentType)
	}
	if captured.Accept != "*/*" {
		t.Errorf("Accept = %q, want */*", captured.Accept)
	}
	if string(captured.Body) != "FAKEZIP" {
		t.Errorf("Body = %q", string(captured.Body))
	}
}

func TestUploadFile_URLMethod(t *testing.T) {
	var captured struct{ Path, ContentType string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Datastores.InWorkspace("topp").UploadFile(context.Background(), "states_shp",
		strings.NewReader("file:///data/states.shp"),
		datastores.UploadOptions{Method: datastores.UploadMethodURL, Extension: "shp"}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if captured.Path != "/rest/workspaces/topp/datastores/states_shp/url.shp" {
		t.Errorf("Path = %q", captured.Path)
	}
	if captured.ContentType != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", captured.ContentType)
	}
}

func TestUploadFile_ExternalMethod(t *testing.T) {
	var captured struct{ Path string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Datastores.InWorkspace("topp").UploadFile(context.Background(), "states_shp",
		strings.NewReader("/srv/geoserver/data/states.shp"),
		datastores.UploadOptions{Method: datastores.UploadMethodExternal, Extension: "shp"}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if captured.Path != "/rest/workspaces/topp/datastores/states_shp/external.shp" {
		t.Errorf("Path = %q", captured.Path)
	}
}

func TestUploadFile_UpdateQuery(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("update")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_ = c.Datastores.InWorkspace("topp").UploadFile(context.Background(), "states_shp",
		strings.NewReader(""), datastores.UploadOptions{Extension: "shp", Update: "overwrite"})
	if captured != "overwrite" {
		t.Errorf("update = %q, want overwrite", captured)
	}
}

func TestUploadFile_ContentTypeOverride(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_ = c.Datastores.InWorkspace("topp").UploadFile(context.Background(), "states_shp",
		strings.NewReader(""),
		datastores.UploadOptions{Extension: "shp", ContentType: "application/octet-stream"})
	if captured != "application/octet-stream" {
		t.Errorf("Content-Type = %q", captured)
	}
}

func TestUploadFile_Validation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	cases := []struct {
		name string
		ws   string
		ds   string
		body io.Reader
		opts datastores.UploadOptions
	}{
		{"empty workspace", "", "states", strings.NewReader(""), datastores.UploadOptions{Extension: "shp"}},
		{"empty name", "topp", "", strings.NewReader(""), datastores.UploadOptions{Extension: "shp"}},
		{"nil body", "topp", "states", nil, datastores.UploadOptions{Extension: "shp"}},
		{"invalid method", "topp", "states", strings.NewReader(""),
			datastores.UploadOptions{Method: datastores.UploadMethod("bogus"), Extension: "shp"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.Datastores.InWorkspace(tc.ws).UploadFile(context.Background(), tc.ds, tc.body, tc.opts)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestUploadFile_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no workspace", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Datastores.InWorkspace("missing").UploadFile(context.Background(), "states",
		strings.NewReader(""), datastores.UploadOptions{Extension: "shp"})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestInWorkspace_Workspace(t *testing.T) {
	c, err := geoserver.New("http://localhost:8080/geoserver")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.Datastores.InWorkspace("topp").Workspace(); got != "topp" {
		t.Fatalf("Workspace() = %q", got)
	}
}
