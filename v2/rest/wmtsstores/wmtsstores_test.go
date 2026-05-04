package wmtsstores_test

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
	"github.com/hishamkaram/geoserver/v2/rest/wmtsstores"
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

func TestList_EmptyStringWireShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"wmtsStores":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.WMTSStores.InWorkspace("topp").List(context.Background(), wmtsstores.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d entries", len(list))
	}
}

func TestList_Populated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/wmtsstores" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"wmtsStores":{"wmtsStore":[
			{"name":"altgs","href":"http://srv/rest/workspaces/topp/wmtsstores/altgs.json"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.WMTSStores.InWorkspace("topp").List(context.Background(), wmtsstores.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "altgs" {
		t.Errorf("list = %+v", list)
	}
}

func TestGet_OK_WrappedShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"wmtsStore":{"name":"altgs","type":"WMS","enabled":true,"capabilitiesURL":"http://upstream/wms?request=GetCapabilities","maxConnections":6}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	s, err := c.WMTSStores.InWorkspace("topp").Get(context.Background(), "altgs")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.Name != "altgs" || s.CapabilitiesURL == "" || s.MaxConnections != 6 {
		t.Errorf("store = %+v", s)
	}
}

func TestCreate_BodyWrapsInWMTSStoreEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, want := range []string{`"wmtsStore":{`, `"name":"new"`, `"capabilitiesURL":"http://upstream/wms"`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.WMTSStores.InWorkspace("topp").Create(context.Background(), &wmtsstores.WMTSStore{
		Name:            "new",
		CapabilitiesURL: "http://upstream/wms",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_RequiresFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	for _, in := range []*wmtsstores.WMTSStore{nil, {}, {Name: "x"}, {CapabilitiesURL: "y"}} {
		if err := c.WMTSStores.InWorkspace("topp").Create(context.Background(), in); err == nil {
			t.Errorf("expected error for %+v", in)
		}
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/workspaces/topp/wmtsstores/altgs" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.WMTSStores.InWorkspace("topp").Delete(context.Background(), "altgs", wmtsstores.DeleteOptions{Recurse: true}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.WMTSStores.InWorkspace("topp").Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
