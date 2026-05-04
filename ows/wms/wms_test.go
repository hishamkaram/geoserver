package wms_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/ows/wms"
)

const minimalCapsXML = `<?xml version="1.0" encoding="UTF-8"?>
<WMT_MS_Capabilities version="1.1.1" updateSequence="42">
  <Service>
    <Name>OGC:WMS</Name>
    <Title>Test GeoServer Web Map Service</Title>
    <KeywordList>
      <Keyword>WMS</Keyword>
      <Keyword>GeoServer</Keyword>
    </KeywordList>
    <OnlineResource xmlns:xlink="http://www.w3.org/1999/xlink"
                    xlink:type="simple" xlink:href="http://example.com/geoserver/wms"/>
    <Fees>NONE</Fees>
    <AccessConstraints>NONE</AccessConstraints>
  </Service>
  <Capability>
    <Request>
      <GetCapabilities>
        <Format>application/vnd.ogc.wms_xml</Format>
        <DCPType>
          <HTTP>
            <Get>
              <OnlineResource xmlns:xlink="http://www.w3.org/1999/xlink"
                              xlink:type="simple" xlink:href="http://example.com/geoserver/wms?"/>
            </Get>
          </HTTP>
        </DCPType>
      </GetCapabilities>
    </Request>
    <Exception>
      <Format>application/vnd.ogc.se_xml</Format>
    </Exception>
    <Layer>
      <Title>Root layer</Title>
      <Layer queryable="1">
        <Title>states</Title>
        <Abstract>USA states</Abstract>
        <SRS>EPSG:4326</SRS>
        <LatLonBoundingBox minx="-130" miny="20" maxx="-65" maxy="50"/>
        <BoundingBox SRS="EPSG:4326" minx="-130" miny="20" maxx="-65" maxy="50"/>
        <Style>
          <Name>polygon</Name>
          <Title>Default polygon style</Title>
          <LegendURL height="20" width="20">
            <Format>image/png</Format>
            <OnlineResource xmlns:xlink="http://www.w3.org/1999/xlink"
                            xlink:type="simple" xlink:href="http://example.com/legend.png"/>
          </LegendURL>
        </Style>
      </Layer>
    </Layer>
  </Capability>
</WMT_MS_Capabilities>`

func TestParseCapabilities_OK(t *testing.T) {
	caps, err := wms.ParseCapabilities(strings.NewReader(minimalCapsXML))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	if caps.Version != "1.1.1" {
		t.Errorf("Version = %q, want 1.1.1", caps.Version)
	}
	if caps.UpdateSequence != "42" {
		t.Errorf("UpdateSequence = %q, want 42", caps.UpdateSequence)
	}
	if caps.Service.Title != "Test GeoServer Web Map Service" {
		t.Errorf("Service.Title = %q", caps.Service.Title)
	}
	if got := len(caps.Service.KeywordList.Keyword); got != 2 {
		t.Errorf("KeywordList: got %d keywords, want 2", got)
	}

	// Layer tree: root has one child layer named "states".
	root := caps.Capability.Layer
	if got := len(root.Layer); got != 1 {
		t.Fatalf("root has %d child layers, want 1", got)
	}
	states := root.Layer[0]
	if states.Title != "states" {
		t.Errorf("child Title = %q", states.Title)
	}
	if states.Queryable != 1 {
		t.Errorf("Queryable = %d, want 1", states.Queryable)
	}
	if got := len(states.SRS); got != 1 || states.SRS[0] == nil || *states.SRS[0] != "EPSG:4326" {
		t.Errorf("SRS = %+v", states.SRS)
	}
	if states.LatLonBoundingBox.MinX != -130 || states.LatLonBoundingBox.MaxY != 50 {
		t.Errorf("LatLonBoundingBox = %+v", states.LatLonBoundingBox)
	}
	if got := len(states.Style); got != 1 || states.Style[0].Name != "polygon" {
		t.Errorf("Style = %+v", states.Style)
	}
}

func TestParseCapabilities_NilReader(t *testing.T) {
	_, err := wms.ParseCapabilities(nil)
	if err == nil {
		t.Fatalf("expected error on nil reader")
	}
}

func TestParseCapabilities_Malformed(t *testing.T) {
	_, err := wms.ParseCapabilities(strings.NewReader("<not-wms/>"))
	if err == nil {
		t.Fatalf("expected error on non-WMS XML")
	}
	// xml.Unmarshal returns "expected element type <WMT_MS_Capabilities> but
	// have <not-wms>". We just check the error is wrapped under our prefix.
	if !strings.Contains(err.Error(), "wms: parse capabilities") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestGetCapabilities_GlobalScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wms" {
			t.Errorf("path = %q, want /wms", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("service") != "wms" || q.Get("request") != "GetCapabilities" {
			t.Errorf("query = %+v", q)
		}
		if q.Get("version") != "1.1.1" {
			t.Errorf("version = %q, want 1.1.1 (default)", q.Get("version"))
		}
		w.Header().Set("Content-Type", "application/vnd.ogc.wms_xml")
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, err := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	caps, err := c.WMS.GetCapabilities(context.Background(), wms.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if caps.Version != "1.1.1" {
		t.Errorf("Version = %q", caps.Version)
	}
}

func TestGetCapabilities_WorkspaceScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topp/wms" {
			t.Errorf("path = %q, want /topp/wms", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.ogc.wms_xml")
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	if _, err := c.WMS.InWorkspace("topp").GetCapabilities(context.Background(),
		wms.GetCapabilitiesOptions{}); err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
}

func TestGetCapabilities_VersionAndUpdateSequence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("version") != "1.3.0" {
			t.Errorf("version = %q, want 1.3.0", q.Get("version"))
		}
		if q.Get("updatesequence") != "42" {
			t.Errorf("updatesequence = %q, want 42", q.Get("updatesequence"))
		}
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, _ = c.WMS.GetCapabilities(context.Background(), wms.GetCapabilitiesOptions{
		Version:        "1.3.0",
		UpdateSequence: "42",
	})
}

func TestGetCapabilities_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such workspace", http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	_, err := c.WMS.InWorkspace("missing").GetCapabilities(context.Background(),
		wms.GetCapabilitiesOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetCapabilities_InternalServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "kaboom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	_, err := c.WMS.GetCapabilities(context.Background(), wms.GetCapabilitiesOptions{})
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("err = %v, want ErrServerError", err)
	}
}

func TestClient_IsGlobal(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	if !c.WMS.IsGlobal() {
		t.Errorf("freshly constructed WMS client should be global")
	}
	scoped := c.WMS.InWorkspace("topp")
	if scoped.IsGlobal() {
		t.Errorf("workspace-scoped client reports IsGlobal=true")
	}
	if scoped.Workspace() != "topp" {
		t.Errorf("Workspace() = %q", scoped.Workspace())
	}
	// Original is unaffected.
	if !c.WMS.IsGlobal() {
		t.Errorf("InWorkspace mutated original")
	}
}
