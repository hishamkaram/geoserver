package layers_test

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
	"github.com/hishamkaram/geoserver/v2/rest/layers"
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

func expectBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:geoserver"))
	if got := r.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization header = %q, want %q", got, want)
	}
}

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.URL.Path != "/rest/workspaces/topp/layers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layers":{"layer":[{"name":"states"},{"name":"counties"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Layers.InWorkspace("topp").List(context.Background(), layers.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "states" {
		t.Fatalf("List = %+v", got)
	}
}

func TestList_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Layers.InWorkspace("topp").List(context.Background(), layers.ListOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList_EmptyWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Layers.InWorkspace("").List(context.Background(), layers.ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "empty workspace") {
		t.Fatalf("expected empty-workspace error, got %v", err)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layers":{"layer":[{"name":"a"},{"name":"b"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for l, err := range c.Layers.InWorkspace("topp").Iter(context.Background(), layers.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, l.Name)
	}
	if len(names) != 2 {
		t.Fatalf("Iter = %v", names)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/workspaces/topp/layers/states" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layer":{
			"name":"states","type":"VECTOR","queryable":true,
			"defaultStyle":{"name":"polygon","href":"http://localhost:8080/geoserver/rest/styles/polygon.json"},
			"styles":{"@class":"linked-hash-set","style":[{"name":"states","href":"http://x/states.json"}]},
			"resource":{"@class":"featureType","name":"topp:states","href":"http://x/states.json"}
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	l, err := c.Layers.InWorkspace("topp").Get(context.Background(), "states")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Name != "states" || l.Type != "VECTOR" || !l.Queryable {
		t.Fatalf("Layer = %+v", l)
	}
	if l.DefaultStyle == nil || l.DefaultStyle.Name != "polygon" {
		t.Fatalf("DefaultStyle = %+v", l.DefaultStyle)
	}
	if l.Styles == nil || l.Styles.Class != "linked-hash-set" || len(l.Styles.Style) != 1 || l.Styles.Style[0].Name != "states" {
		t.Fatalf("Styles = %+v", l.Styles)
	}
	if l.Resource == nil || l.Resource.Class != "featureType" || l.Resource.Name != "topp:states" {
		t.Fatalf("Resource = %+v", l.Resource)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Layers.InWorkspace("topp").Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/workspaces/topp/layers/states" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"layer":`) || !strings.Contains(s, `"defaultStyle":{"name":"line"}`) {
			t.Errorf("body = %s", s)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Layers.InWorkspace("topp").Update(context.Background(), "states", &layers.Layer{
		DefaultStyle: &layers.Ref{Name: "line"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NilLayer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Layers.InWorkspace("topp").Update(context.Background(), "states", nil)
	if err == nil || !strings.Contains(err.Error(), "nil layer") {
		t.Fatalf("expected nil-layer error, got %v", err)
	}
}

func TestDelete_RecurseQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/workspaces/topp/layers/states" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("recurse") != "true" {
			t.Errorf("recurse = %q", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Layers.InWorkspace("topp").Delete(context.Background(), "states",
		layers.DeleteOptions{Recurse: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Layers.InWorkspace("topp").Delete(context.Background(), "states",
		layers.DeleteOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

func TestListStyles_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/rest/workspaces/topp/layers/states/styles" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"styles":{"style":[
            {"name":"polygon","href":"http://example.com/styles/polygon.json"},
            {"name":"line","href":"http://example.com/styles/line.json"}
        ]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Layers.InWorkspace("topp").ListStyles(context.Background(), "states")
	if err != nil {
		t.Fatalf("ListStyles: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 styles, got %d", len(got))
	}
	if got[0].Name != "polygon" || got[1].Name != "line" {
		t.Errorf("names = %+v", got)
	}
}

func TestListStyles_EmptyValidation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	if _, err := c.Layers.InWorkspace("").ListStyles(context.Background(), "states"); err == nil {
		t.Errorf("expected error for empty workspace")
	}
	if _, err := c.Layers.InWorkspace("topp").ListStyles(context.Background(), ""); err == nil {
		t.Errorf("expected error for empty layer name")
	}
}

func TestListStyles_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such layer", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Layers.InWorkspace("topp").ListStyles(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAddStyle_OK(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/rest/workspaces/topp/layers/states/styles" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		// No `default` query when opts.Default=false.
		if r.URL.Query().Get("default") != "" {
			t.Errorf("unexpected default=%q", r.URL.Query().Get("default"))
		}
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Layers.InWorkspace("topp").AddStyle(context.Background(), "states", "polygon",
		layers.AddStyleOptions{}); err != nil {
		t.Fatalf("AddStyle: %v", err)
	}
	if !strings.Contains(capturedBody, `"name":"polygon"`) {
		t.Errorf("body missing style name: %q", capturedBody)
	}
	if !strings.Contains(capturedBody, `"style":`) {
		t.Errorf("body missing style envelope: %q", capturedBody)
	}
}

func TestAddStyle_DefaultFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("default") != "true" {
			t.Errorf("default = %q, want true", r.URL.Query().Get("default"))
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Layers.InWorkspace("topp").AddStyle(context.Background(), "states", "polygon",
		layers.AddStyleOptions{Default: true}); err != nil {
		t.Fatalf("AddStyle: %v", err)
	}
}

func TestAddStyle_EmptyValidation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	cases := []struct {
		name             string
		ws, layer, style string
	}{
		{"empty workspace", "", "states", "polygon"},
		{"empty layer", "topp", "", "polygon"},
		{"empty style", "topp", "states", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.Layers.InWorkspace(tc.ws).AddStyle(context.Background(),
				tc.layer, tc.style, layers.AddStyleOptions{})
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestAddStyle_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "style not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Layers.InWorkspace("topp").AddStyle(context.Background(), "states", "missing-style",
		layers.AddStyleOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGet_URLEscaping(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layer":{"name":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Layers.InWorkspace("ws*1").Get(context.Background(), "ly*2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURI, "ws%2A1") || !strings.Contains(capturedURI, "ly%2A2") {
		t.Fatalf("expected single-encoded segments, got %q", capturedURI)
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}
