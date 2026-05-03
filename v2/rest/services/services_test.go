package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/services"
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

// ===== WMS =====

func TestWMS_Get_Global(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/services/wms/settings" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"wms":{
            "enabled":true,
            "name":"WMS",
            "title":"My WMS",
            "maxRenderingTime":120,
            "maxBuffer":50,
            "watermark":{"enabled":true,"position":"BOT_RIGHT","transparency":80},
            "interpolation":"Bilinear"
        }}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Enabled || got.Title != "My WMS" {
		t.Errorf("ServiceInfo not decoded: %+v", got.ServiceInfo)
	}
	if got.MaxRenderingTime != 120 || got.MaxBuffer != 50 {
		t.Errorf("WMS-only fields: MaxRenderingTime=%d MaxBuffer=%d", got.MaxRenderingTime, got.MaxBuffer)
	}
	if got.Watermark == nil || got.Watermark.Position != "BOT_RIGHT" || got.Watermark.Transparency != 80 {
		t.Errorf("Watermark = %+v", got.Watermark)
	}
	if got.Interpolation != "Bilinear" {
		t.Errorf("Interpolation = %q", got.Interpolation)
	}
}

func TestWMS_Update_Global_Envelope(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method = %q", r.Method)
		}
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Services.WMS().Update(context.Background(), &services.WMSSettings{
		ServiceInfo:      services.ServiceInfo{Enabled: true, Title: "T"},
		MaxRenderingTime: 60,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !strings.HasPrefix(string(captured), `{"wms":`) {
		t.Errorf("body envelope wrong: %q", string(captured))
	}
	if !strings.Contains(string(captured), `"maxRenderingTime":60`) {
		t.Errorf("body missing maxRenderingTime: %q", string(captured))
	}
}

func TestWMS_Update_NilSettings(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	if err := c.Services.WMS().Update(context.Background(), nil); err == nil {
		t.Errorf("expected error for nil settings")
	}
}

func TestWMS_InWorkspace_GetUpdateDelete(t *testing.T) {
	gets := 0
	puts := 0
	deletes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/rest/services/wms/workspaces/topp/settings"
		if r.URL.Path != want {
			t.Errorf("Path = %q, want %q", r.URL.Path, want)
		}
		switch r.Method {
		case http.MethodGet:
			gets++
			_, _ = io.WriteString(w, `{"wms":{"name":"WMS","title":"topp WMS"}}`)
		case http.MethodPut:
			puts++
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			deletes++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ws := c.Services.WMS().InWorkspace("topp")

	if got, err := ws.Get(context.Background()); err != nil || got.Title != "topp WMS" {
		t.Fatalf("Get: %v %+v", err, got)
	}
	if err := ws.Update(context.Background(), &services.WMSSettings{
		ServiceInfo: services.ServiceInfo{Title: "new"},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := ws.Delete(context.Background()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if gets != 1 || puts != 1 || deletes != 1 {
		t.Errorf("verb counts = G%d P%d D%d", gets, puts, deletes)
	}
}

func TestWMS_InWorkspace_Validation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	emptyWS := c.Services.WMS().InWorkspace("")

	if _, err := emptyWS.Get(context.Background()); err == nil {
		t.Errorf("expected error from Get with empty workspace")
	}
	if err := emptyWS.Update(context.Background(), &services.WMSSettings{}); err == nil {
		t.Errorf("expected error from Update with empty workspace")
	}
	if err := emptyWS.Delete(context.Background()); err == nil {
		t.Errorf("expected error from Delete with empty workspace")
	}
}

func TestWMS_Get_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no override", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Services.WMS().InWorkspace("missing").Get(context.Background())
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// ===== WFS =====

func TestWFS_Get_Global(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/services/wfs/settings" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"wfs":{
            "name":"WFS",
            "maxFeatures":1000,
            "serviceLevel":"COMPLETE",
            "featureBounding":true,
            "hitsIgnoreMaxFeatures":false
        }}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WFS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.MaxFeatures != 1000 || got.ServiceLevel != "COMPLETE" || !got.FeatureBounding {
		t.Errorf("WFS fields wrong: %+v", got)
	}
}

func TestWFS_Update_Envelope(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_ = c.Services.WFS().Update(context.Background(), &services.WFSSettings{
		ServiceInfo:  services.ServiceInfo{Title: "WFS"},
		MaxFeatures:  500,
		ServiceLevel: "BASIC",
	})
	if !strings.HasPrefix(string(captured), `{"wfs":`) {
		t.Errorf("body envelope wrong: %q", string(captured))
	}
	if !strings.Contains(string(captured), `"maxFeatures":500`) {
		t.Errorf("body missing maxFeatures: %q", string(captured))
	}
	if !strings.Contains(string(captured), `"serviceLevel":"BASIC"`) {
		t.Errorf("body missing serviceLevel: %q", string(captured))
	}
}

// ===== WCS =====

func TestWCS_Get_Global(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/services/wcs/settings" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		// MaxInputMemory and MaxOutputMemory are integers despite the
		// upstream YAML doc-bug; this fixture sends them as integers.
		_, _ = io.WriteString(w, `{"wcs":{
            "name":"WCS",
            "gmlPrefixing":true,
            "latLon":false,
            "maxInputMemory":1048576,
            "maxOutputMemory":2097152
        }}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WCS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.GMLPrefixing || got.LatLon {
		t.Errorf("bool fields = %+v", got)
	}
	if got.MaxInputMemory != 1_048_576 || got.MaxOutputMemory != 2_097_152 {
		t.Errorf("memory fields: in=%d out=%d", got.MaxInputMemory, got.MaxOutputMemory)
	}
}

// ===== WMTS =====

func TestWMTS_Get_Global(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/services/wmts/settings" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"wmts":{"name":"WMTS","title":"My WMTS","verbose":true}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMTS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "My WMTS" || !got.Verbose {
		t.Errorf("WMTS = %+v", got)
	}
}

func TestWMTS_Update_Envelope(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_ = c.Services.WMTS().Update(context.Background(), &services.WMTSSettings{
		ServiceInfo: services.ServiceInfo{Title: "T"},
	})
	if !strings.HasPrefix(string(captured), `{"wmts":`) {
		t.Errorf("body envelope wrong: %q", string(captured))
	}
}

// ===== Wire-format quirks =====

func TestWireFormat_Versions_Array(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"wms":{"versions":{"org.geotools.util.Version":[
			{"version":"1.1.1"},{"version":"1.3.0"}
		]}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Versions == nil || len(got.Versions.List) != 2 {
		t.Fatalf("Versions.List = %+v, want 2 entries", got.Versions)
	}
	if got.Versions.List[0] != "1.1.1" || got.Versions.List[1] != "1.3.0" {
		t.Errorf("Versions = %+v", got.Versions.List)
	}
}

func TestWireFormat_Versions_SingleObjectCollapse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"wmts":{"versions":{"org.geotools.util.Version":
			{"version":"1.0.0"}
		}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMTS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Versions == nil || len(got.Versions.List) != 1 || got.Versions.List[0] != "1.0.0" {
		t.Fatalf("single-object collapse not handled: %+v", got.Versions)
	}
}

func TestWireFormat_Keywords_Array(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"wms":{"keywords":{"string":["WMS","GEOSERVER"]}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Keywords == nil || len(got.Keywords.Strings) != 2 {
		t.Fatalf("Keywords.Strings = %+v, want 2 entries", got.Keywords)
	}
}

func TestWireFormat_Keywords_SingleStringCollapse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"wmts":{"keywords":{"string":"WMTS"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMTS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Keywords == nil || len(got.Keywords.Strings) != 1 || got.Keywords.Strings[0] != "WMTS" {
		t.Fatalf("single-string collapse not handled: %+v", got.Keywords)
	}
}

func TestWireFormat_MetadataLink_EmptyString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GeoServer sends `"metadataLink": ""` when no link is configured.
		_, _ = io.WriteString(w, `{"wms":{"name":"WMS","metadataLink":""}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Services.WMS().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Implementation note: the field is *MetadataLink and the empty-
	// string form leaves a non-nil pointer to a zero struct. Both the
	// "nil pointer" and "non-nil zero struct" cases are documented as
	// equivalent to "unset"; check for empty fields rather than nil.
	if got.MetadataLink != nil && (got.MetadataLink.Type != "" || got.MetadataLink.Content != "") {
		t.Errorf("expected empty MetadataLink, got %+v", got.MetadataLink)
	}
}

func TestWireFormat_Versions_Marshal_CanonicalArray(t *testing.T) {
	v := services.Versions{List: []string{"2.0.0"}}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), `"org.geotools.util.Version":[`) {
		t.Errorf("Marshal didn't emit class-name wrapper: %s", string(out))
	}
	if !strings.Contains(string(out), `"version":"2.0.0"`) {
		t.Errorf("Marshal didn't include version: %s", string(out))
	}
}

// ===== Cross-cutting =====

func TestServerError_PassThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Services.WMS().Get(context.Background())
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("err = %v, want ErrServerError", err)
	}
}
