package wcs_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/ows/wcs"
)

const minimalCapsXML = `<?xml version="1.0" encoding="UTF-8"?>
<wcs:Capabilities version="2.0.1" updateSequence="42"
    xmlns:wcs="http://www.opengis.net/wcs/2.0"
    xmlns:ows="http://www.opengis.net/ows/2.0"
    xmlns:xlink="http://www.w3.org/1999/xlink">
  <ows:ServiceIdentification>
    <ows:Title>Test WCS</ows:Title>
    <ows:Abstract>Integration-test fixture</ows:Abstract>
    <ows:Keywords>
      <ows:Keyword>WCS</ows:Keyword>
    </ows:Keywords>
    <ows:ServiceType>OGC WCS</ows:ServiceType>
    <ows:ServiceTypeVersion>2.0.1</ows:ServiceTypeVersion>
    <ows:Profile>http://www.opengis.net/spec/WCS_service-extension_crs/1.0/conf/crs</ows:Profile>
    <ows:Fees>NONE</ows:Fees>
    <ows:AccessConstraints>NONE</ows:AccessConstraints>
  </ows:ServiceIdentification>
  <ows:ServiceProvider>
    <ows:ProviderName>Acme</ows:ProviderName>
    <ows:ProviderSite xlink:type="simple" xlink:href="http://example.com/"/>
  </ows:ServiceProvider>
  <ows:OperationsMetadata>
    <ows:Operation name="GetCapabilities">
      <ows:DCP>
        <ows:HTTP>
          <ows:Get xlink:href="http://example.com/geoserver/wcs?"/>
        </ows:HTTP>
      </ows:DCP>
    </ows:Operation>
    <ows:Operation name="GetCoverage">
      <ows:DCP>
        <ows:HTTP>
          <ows:Get xlink:href="http://example.com/geoserver/wcs?"/>
          <ows:Post xlink:href="http://example.com/geoserver/wcs"/>
        </ows:HTTP>
      </ows:DCP>
    </ows:Operation>
  </ows:OperationsMetadata>
  <wcs:ServiceMetadata>
    <wcs:formatSupported>image/tiff</wcs:formatSupported>
    <wcs:formatSupported>image/png</wcs:formatSupported>
    <wcs:Extension>
      <wcs:crsSupported>http://www.opengis.net/def/crs/EPSG/0/4326</wcs:crsSupported>
      <wcs:crsSupported>http://www.opengis.net/def/crs/EPSG/0/3857</wcs:crsSupported>
    </wcs:Extension>
  </wcs:ServiceMetadata>
  <wcs:Contents>
    <wcs:CoverageSummary>
      <wcs:CoverageId>nurc__world_dem</wcs:CoverageId>
      <wcs:CoverageSubtype>RectifiedGridCoverage</wcs:CoverageSubtype>
    </wcs:CoverageSummary>
  </wcs:Contents>
</wcs:Capabilities>`

func TestParseCapabilities_OK(t *testing.T) {
	caps, err := wcs.ParseCapabilities(strings.NewReader(minimalCapsXML))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	if caps.Version != "2.0.1" {
		t.Errorf("Version = %q, want 2.0.1", caps.Version)
	}
	si := caps.ServiceIdentification
	if si.Title != "Test WCS" {
		t.Errorf("Title = %q", si.Title)
	}
	if got := len(si.Profiles); got != 1 {
		t.Errorf("Profiles: got %d, want 1", got)
	}
	if got := len(caps.ServiceMetadata.Formats); got != 2 {
		t.Errorf("Formats: got %d, want 2", got)
	}
	if got := len(caps.ServiceMetadata.CRS); got != 2 {
		t.Errorf("CRS: got %d, want 2", got)
	}
	if got := len(caps.Contents.CoverageSummary); got != 1 {
		t.Fatalf("CoverageSummary: got %d, want 1", got)
	}
	cov := caps.Contents.CoverageSummary[0]
	if cov.CoverageID != "nurc__world_dem" {
		t.Errorf("CoverageID = %q", cov.CoverageID)
	}
	if cov.CoverageSubtype != "RectifiedGridCoverage" {
		t.Errorf("CoverageSubtype = %q", cov.CoverageSubtype)
	}
}

func TestParseCapabilities_NilReader(t *testing.T) {
	if _, err := wcs.ParseCapabilities(nil); err == nil {
		t.Fatalf("expected error on nil reader")
	}
}

func TestParseCapabilities_Malformed(t *testing.T) {
	_, err := wcs.ParseCapabilities(strings.NewReader("<not-wcs/>"))
	if err == nil {
		t.Fatalf("expected error on non-WCS XML")
	}
	if !strings.Contains(err.Error(), "wcs: parse capabilities") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestGetCapabilities_GlobalDefaultVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wcs" {
			t.Errorf("path = %q, want /wcs", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("service") != "WCS" {
			t.Errorf("service = %q (GeoServer WCS endpoint requires uppercase)", q.Get("service"))
		}
		if q.Get("version") != "2.0.1" {
			t.Errorf("version = %q, want 2.0.1 (default)", q.Get("version"))
		}
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	caps, err := c.WCS.GetCapabilities(context.Background(), wcs.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if caps.Version != "2.0.1" {
		t.Errorf("Version = %q", caps.Version)
	}
}

func TestGetCapabilities_WorkspaceScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nurc/wcs" {
			t.Errorf("path = %q, want /nurc/wcs", r.URL.Path)
		}
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	if _, err := c.WCS.InWorkspace("nurc").GetCapabilities(context.Background(),
		wcs.GetCapabilitiesOptions{}); err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
}

func TestGetCapabilities_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such workspace", http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, err := c.WCS.InWorkspace("missing").GetCapabilities(context.Background(),
		wcs.GetCapabilitiesOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestClient_IsGlobal(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	if !c.WCS.IsGlobal() {
		t.Errorf("freshly constructed WCS client should be global")
	}
	scoped := c.WCS.InWorkspace("nurc")
	if scoped.IsGlobal() {
		t.Errorf("workspace-scoped client reports IsGlobal=true")
	}
	if scoped.Workspace() != "nurc" {
		t.Errorf("Workspace() = %q", scoped.Workspace())
	}
	if !c.WCS.IsGlobal() {
		t.Errorf("InWorkspace mutated original")
	}
}
