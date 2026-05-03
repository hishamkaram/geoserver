package layergroups_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/layergroups"
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

// ---- Mixed-shape Unmarshal tests ----

func TestPublished_UnmarshalArray(t *testing.T) {
	var p layergroups.Published
	if err := json.Unmarshal([]byte(`[{"@type":"layer","name":"states"},{"@type":"layer","name":"counties"}]`), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p) != 2 || p[0].Name != "states" || p[1].Name != "counties" {
		t.Fatalf("Published = %+v", p)
	}
}

func TestPublished_UnmarshalSingleObject(t *testing.T) {
	var p layergroups.Published
	if err := json.Unmarshal([]byte(`{"@type":"layer","name":"only_one"}`), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p) != 1 || p[0].Name != "only_one" || p[0].Type != "layer" {
		t.Fatalf("Published = %+v", p)
	}
}

func TestStyles_MixedArray(t *testing.T) {
	// One layer uses default ("" string), the other has an explicit style object.
	var s layergroups.Styles
	if err := json.Unmarshal([]byte(`{"style":["",{"name":"polygon","href":"http://x/polygon.json"}]}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.Style) != 2 {
		t.Fatalf("Style = %+v", s.Style)
	}
	if s.Style[0].Name != "" {
		t.Fatalf("Style[0] = %+v, want empty Name (default style sentinel)", s.Style[0])
	}
	if s.Style[1].Name != "polygon" || s.Style[1].Href == "" {
		t.Fatalf("Style[1] = %+v", s.Style[1])
	}
}

func TestStyles_NamedString(t *testing.T) {
	// String entries that are non-empty (named-but-no-href shape)
	var s layergroups.Styles
	if err := json.Unmarshal([]byte(`{"style":["polygon","line"]}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.Style) != 2 || s.Style[0].Name != "polygon" || s.Style[1].Name != "line" {
		t.Fatalf("Style = %+v", s.Style)
	}
}

// ---- HTTP CRUD tests ----

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBasicAuth(t, r)
		if r.URL.Path != "/rest/workspaces/topp/layergroups" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layerGroups":{"layerGroup":[{"name":"tasmania"},{"name":"spearfish"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.LayerGroups.InWorkspace("topp").List(context.Background(), layergroups.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "tasmania" || got[1].Name != "spearfish" {
		t.Fatalf("List = %+v", got)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layerGroups":{"layerGroup":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var names []string
	for g, err := range c.LayerGroups.InWorkspace("topp").Iter(context.Background(), layergroups.ListOptions{}) {
		if err != nil {
			t.Fatalf("iter error: %v", err)
		}
		names = append(names, g.Name)
	}
	if len(names) != 3 {
		t.Fatalf("Iter = %v", names)
	}
}

func TestGet_OK_SingleMember(t *testing.T) {
	// Single-member group: Published comes back as object, not array.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layerGroup":{
			"name":"solo","mode":"SINGLE",
			"publishables":{"published":{"@type":"layer","name":"states","href":"http://x/states.json"}},
			"styles":{"style":[""]}
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	g, err := c.LayerGroups.InWorkspace("topp").Get(context.Background(), "solo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name != "solo" || g.Mode != "SINGLE" {
		t.Fatalf("LayerGroup = %+v", g)
	}
	if len(g.Publishables.Published) != 1 || g.Publishables.Published[0].Name != "states" {
		t.Fatalf("Published = %+v", g.Publishables.Published)
	}
	if len(g.Styles.Style) != 1 || g.Styles.Style[0].Name != "" {
		t.Fatalf("Styles = %+v", g.Styles)
	}
}

func TestGet_OK_MultipleMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layerGroup":{
			"name":"tasmania","mode":"SINGLE",
			"publishables":{"published":[
				{"@type":"layer","name":"states"},
				{"@type":"layer","name":"counties"}
			]},
			"styles":{"style":["",{"name":"polygon"}]}
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	g, err := c.LayerGroups.InWorkspace("topp").Get(context.Background(), "tasmania")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Publishables.Published) != 2 {
		t.Fatalf("Published = %+v", g.Publishables.Published)
	}
	if len(g.Styles.Style) != 2 || g.Styles.Style[1].Name != "polygon" {
		t.Fatalf("Styles = %+v", g.Styles)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.LayerGroups.InWorkspace("topp").Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/workspaces/topp/layergroups" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"layerGroup":`) || !strings.Contains(s, `"name":"tasmania"`) {
			t.Errorf("body = %s", s)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.LayerGroups.InWorkspace("topp").Create(context.Background(), &layergroups.LayerGroup{
		Name: "tasmania",
		Mode: "SINGLE",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.LayerGroups.InWorkspace("topp").Create(context.Background(), &layergroups.LayerGroup{
		Name: "dup",
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_NilGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.LayerGroups.InWorkspace("topp").Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil layer group") {
		t.Fatalf("expected nil-group error, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/workspaces/topp/layergroups/tasmania" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"title":"Updated"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.LayerGroups.InWorkspace("topp").Update(context.Background(), "tasmania",
		&layergroups.LayerGroup{Title: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NoQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/workspaces/topp/layergroups/tasmania" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		// LayerGroup delete in v1 doesn't pass ?recurse — confirm v2 doesn't either.
		if r.URL.Query().Get("recurse") != "" {
			t.Errorf("unexpected recurse query: %q", r.URL.Query().Get("recurse"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.LayerGroups.InWorkspace("topp").Delete(context.Background(), "tasmania")
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
	err := c.LayerGroups.InWorkspace("topp").Delete(context.Background(), "tasmania")
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

func TestGet_URLEscaping(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"layerGroup":{"name":"x"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.LayerGroups.InWorkspace("ws*1").Get(context.Background(), "lg*2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURI, "ws%2A1") || !strings.Contains(capturedURI, "lg%2A2") {
		t.Fatalf("expected single-encoded segments, got %q", capturedURI)
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}
