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

const minimalCoverageDescriptionsXML = `<?xml version="1.0" encoding="UTF-8"?>
<wcs:CoverageDescriptions
    xmlns:wcs="http://www.opengis.net/wcs/2.0"
    xmlns:gml="http://www.opengis.net/gml/3.2"
    xmlns:gmlcov="http://www.opengis.net/gmlcov/1.0"
    xmlns:swe="http://www.opengis.net/swe/2.0">
  <wcs:CoverageDescription gml:id="nurc__Arc_Sample">
    <wcs:CoverageId>nurc__Arc_Sample</wcs:CoverageId>
    <gml:boundedBy>
      <gml:Envelope srsName="http://www.opengis.net/def/crs/EPSG/0/4326"
                    axisLabels="Lat Long"
                    uomLabels="deg deg"
                    srsDimension="2">
        <gml:lowerCorner>-90.0 -180.0</gml:lowerCorner>
        <gml:upperCorner>90.0 180.0</gml:upperCorner>
      </gml:Envelope>
    </gml:boundedBy>
    <gml:domainSet>
      <gml:RectifiedGrid dimension="2" srsName="http://www.opengis.net/def/crs/EPSG/0/4326">
        <gml:limits>
          <gml:GridEnvelope>
            <gml:low>0 0</gml:low>
            <gml:high>1023 511</gml:high>
          </gml:GridEnvelope>
        </gml:limits>
        <gml:axisLabels>i j</gml:axisLabels>
      </gml:RectifiedGrid>
    </gml:domainSet>
    <gmlcov:rangeType>
      <swe:DataRecord>
        <swe:field name="GRAY_INDEX">
          <swe:Quantity>
            <swe:description>GRAY_INDEX</swe:description>
            <swe:uom code="W.m-2.Sr-1"/>
          </swe:Quantity>
        </swe:field>
      </swe:DataRecord>
    </gmlcov:rangeType>
    <wcs:ServiceParameters>
      <wcs:CoverageSubtype>RectifiedGridCoverage</wcs:CoverageSubtype>
      <wcs:nativeFormat>image/tiff</wcs:nativeFormat>
    </wcs:ServiceParameters>
  </wcs:CoverageDescription>
</wcs:CoverageDescriptions>`

func TestParseCoverageDescriptions_OK(t *testing.T) {
	descs, err := wcs.ParseCoverageDescriptions(strings.NewReader(minimalCoverageDescriptionsXML))
	if err != nil {
		t.Fatalf("ParseCoverageDescriptions: %v", err)
	}
	if got := len(descs.CoverageDescription); got != 1 {
		t.Fatalf("CoverageDescription: got %d, want 1", got)
	}
	d := descs.CoverageDescription[0]
	if d.CoverageID != "nurc__Arc_Sample" {
		t.Errorf("CoverageID = %q", d.CoverageID)
	}
	if d.BoundedBy.Envelope.LowerCorner != "-90.0 -180.0" {
		t.Errorf("LowerCorner = %q", d.BoundedBy.Envelope.LowerCorner)
	}
	if d.BoundedBy.Envelope.SrsName != "http://www.opengis.net/def/crs/EPSG/0/4326" {
		t.Errorf("SrsName = %q", d.BoundedBy.Envelope.SrsName)
	}
	if d.DomainSet.RectifiedGrid.Limits.GridEnvelope.High != "1023 511" {
		t.Errorf("GridEnvelope.High = %q", d.DomainSet.RectifiedGrid.Limits.GridEnvelope.High)
	}
	if got := len(d.RangeType.DataRecord.Field); got != 1 {
		t.Fatalf("Fields: got %d, want 1", got)
	}
	field := d.RangeType.DataRecord.Field[0]
	if field.Name != "GRAY_INDEX" {
		t.Errorf("Field.Name = %q", field.Name)
	}
	if field.Quantity.Uom.Code != "W.m-2.Sr-1" {
		t.Errorf("Field.Uom.Code = %q", field.Quantity.Uom.Code)
	}
	if d.ServiceParameters.CoverageSubtype != "RectifiedGridCoverage" {
		t.Errorf("CoverageSubtype = %q", d.ServiceParameters.CoverageSubtype)
	}
	if d.ServiceParameters.NativeFormat != "image/tiff" {
		t.Errorf("NativeFormat = %q", d.ServiceParameters.NativeFormat)
	}
}

func TestParseCoverageDescriptions_NilReader(t *testing.T) {
	if _, err := wcs.ParseCoverageDescriptions(nil); err == nil {
		t.Fatalf("expected error on nil reader")
	}
}

func TestParseCoverageDescriptions_Malformed(t *testing.T) {
	_, err := wcs.ParseCoverageDescriptions(strings.NewReader("<not-wcs/>"))
	if err == nil {
		t.Fatalf("expected error on non-WCS XML")
	}
	if !strings.Contains(err.Error(), "wcs: parse coverage descriptions") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestDescribeCoverage_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("service") != "WCS" {
			t.Errorf("service = %q (must be uppercase)", q.Get("service"))
		}
		if q.Get("request") != "DescribeCoverage" {
			t.Errorf("request = %q", q.Get("request"))
		}
		if q.Get("coverageId") != "nurc__Arc_Sample" {
			t.Errorf("coverageId = %q", q.Get("coverageId"))
		}
		_, _ = io.WriteString(w, minimalCoverageDescriptionsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	descs, err := c.WCS.DescribeCoverage(context.Background(),
		wcs.DescribeCoverageOptions{CoverageIDs: []string{"nurc__Arc_Sample"}})
	if err != nil {
		t.Fatalf("DescribeCoverage: %v", err)
	}
	if got := len(descs.CoverageDescription); got != 1 {
		t.Errorf("CoverageDescription: got %d, want 1", got)
	}
}

func TestDescribeCoverage_EmptyIDs(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	_, err := c.WCS.DescribeCoverage(context.Background(), wcs.DescribeCoverageOptions{})
	if err == nil {
		t.Fatalf("expected error for empty CoverageIDs")
	}
	if !strings.Contains(err.Error(), "empty CoverageIDs") {
		t.Errorf("error message = %q", err.Error())
	}
}

func TestDescribeCoverage_MultipleIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("coverageId") != "a,b,c" {
			t.Errorf("coverageId = %q", r.URL.Query().Get("coverageId"))
		}
		_, _ = io.WriteString(w, minimalCoverageDescriptionsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, _ = c.WCS.DescribeCoverage(context.Background(),
		wcs.DescribeCoverageOptions{CoverageIDs: []string{"a", "b", "c"}})
}

func TestDescribeCoverage_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such coverage", http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, err := c.WCS.DescribeCoverage(context.Background(),
		wcs.DescribeCoverageOptions{CoverageIDs: []string{"missing"}})
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
