package wfstransforms_test

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
	"github.com/hishamkaram/geoserver/v2/rest/wfstransforms"
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

func TestList_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/services/wfs/transforms" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"transforms":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.WFSTransforms.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty, got %d", len(list))
	}
}

func TestList_Populated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"transforms":{"transform":[
			{"name":"to-html","href":"http://srv/rest/services/wfs/transforms/to-html.json"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.WFSTransforms.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "to-html" {
		t.Errorf("list = %+v", list)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"transform":{"name":"test1","sourceFormat":"text/xml; subtype=gml/2.1.2","outputFormat":"text/html","xslt":"test1.xslt"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	tr, err := c.WFSTransforms.Get(context.Background(), "test1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if tr.Name != "test1" || tr.OutputFormat != "text/html" || tr.XSLT != "test1.xslt" {
		t.Errorf("transform = %+v", tr)
	}
}

func TestGet_NotFound(t *testing.T) {
	// Endpoint isn't installed by default — extension absent path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.WFSTransforms.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreate_BodyWrapsInTransformEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, want := range []string{`"transform":{`, `"name":"to-html"`, `"sourceFormat":"text/xml`, `"outputFormat":"text/html"`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.WFSTransforms.Create(context.Background(), &wfstransforms.Transform{
		Name:         "to-html",
		SourceFormat: "text/xml; subtype=gml/2.1.2",
		OutputFormat: "text/html",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreateWithXSLT_QueryParamsAndContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "to-html" {
			t.Errorf("name = %q", r.URL.Query().Get("name"))
		}
		if r.URL.Query().Get("sourceFormat") != "text/xml; subtype=gml/2.1.2" {
			t.Errorf("sourceFormat = %q", r.URL.Query().Get("sourceFormat"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/xslt+xml" {
			t.Errorf("Content-Type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "<xsl:") {
			t.Errorf("body should contain XSLT; got %s", body)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	xslt := `<?xml version="1.0"?><xsl:stylesheet/>`
	c := newTestClient(t, srv)
	err := c.WFSTransforms.CreateWithXSLT(context.Background(), strings.NewReader(xslt), wfstransforms.CreateWithXSLTOptions{
		Name:         "to-html",
		SourceFormat: "text/xml; subtype=gml/2.1.2",
		OutputFormat: "text/html",
	})
	if err != nil {
		t.Fatalf("CreateWithXSLT: %v", err)
	}
}

func TestPutXSLT_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/services/wfs/transforms/to-html" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/xslt+xml" {
			t.Errorf("Content-Type = %q", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.WFSTransforms.PutXSLT(context.Background(), "to-html", strings.NewReader("<x/>")); err != nil {
		t.Fatalf("PutXSLT: %v", err)
	}
}

func TestDelete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/services/wfs/transforms/to-html" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.WFSTransforms.Delete(context.Background(), "to-html"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
