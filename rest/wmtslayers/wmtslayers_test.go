package wmtslayers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/wmtslayers"
)

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

func TestList_Workspace_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/wmtslayers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"wmtsLayers":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.WMTSLayers.InWorkspace("topp").List(context.Background(), wmtslayers.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty, got %d", len(list))
	}
}

func TestList_Store_Populated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/wmtsstores/altgs/wmtslayers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"wmtsLayers":{"wmtsLayer":[
			{"name":"dem","href":"http://srv/rest/workspaces/topp/wmtsstores/altgs/wmtslayers/dem.json"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.WMTSLayers.InWorkspace("topp").InStore("altgs").List(context.Background(), wmtslayers.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "dem" {
		t.Errorf("list = %+v", list)
	}
}

func TestGet_Store_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"wmtsLayer":{"name":"dem","nativeName":"usgs:dem","srs":"EPSG:4326","enabled":true}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	l, err := c.WMTSLayers.InWorkspace("topp").InStore("altgs").Get(context.Background(), "dem")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if l.Name != "dem" || l.NativeName != "usgs:dem" || l.SRS != "EPSG:4326" {
		t.Errorf("layer = %+v", l)
	}
}

func TestCreate_BodyWrapsInWMTSLayerEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, want := range []string{`"wmtsLayer":{`, `"name":"dem"`, `"nativeName":"usgs:dem"`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.WMTSLayers.InWorkspace("topp").InStore("altgs").Create(context.Background(), &wmtslayers.WMTSLayer{
		Name:       "dem",
		NativeName: "usgs:dem",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.WMTSLayers.InWorkspace("topp").InStore("altgs").Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
