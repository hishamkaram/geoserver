package wfs_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/ows/wfs"
)

const minimalCapsXML = `<?xml version="1.0" encoding="UTF-8"?>
<wfs:WFS_Capabilities version="2.0.0" updateSequence="42"
    xmlns:wfs="http://www.opengis.net/wfs/2.0"
    xmlns:ows="http://www.opengis.net/ows/1.1"
    xmlns:xlink="http://www.w3.org/1999/xlink">
  <ows:ServiceIdentification>
    <ows:Title>Test WFS</ows:Title>
    <ows:Abstract>Integration-test fixture</ows:Abstract>
    <ows:Keywords>
      <ows:Keyword>WFS</ows:Keyword>
      <ows:Keyword>GeoServer</ows:Keyword>
    </ows:Keywords>
    <ows:ServiceType>WFS</ows:ServiceType>
    <ows:ServiceTypeVersion>2.0.0</ows:ServiceTypeVersion>
    <ows:ServiceTypeVersion>1.1.0</ows:ServiceTypeVersion>
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
          <ows:Get xlink:href="http://example.com/geoserver/wfs?"/>
        </ows:HTTP>
      </ows:DCP>
    </ows:Operation>
    <ows:Operation name="GetFeature">
      <ows:DCP>
        <ows:HTTP>
          <ows:Get xlink:href="http://example.com/geoserver/wfs?"/>
          <ows:Post xlink:href="http://example.com/geoserver/wfs"/>
        </ows:HTTP>
      </ows:DCP>
    </ows:Operation>
  </ows:OperationsMetadata>
  <wfs:FeatureTypeList>
    <wfs:FeatureType>
      <wfs:Name>topp:states</wfs:Name>
      <wfs:Title>USA States</wfs:Title>
      <wfs:Abstract>Polygons for USA states</wfs:Abstract>
      <wfs:Keywords>
        <wfs:Keyword>states</wfs:Keyword>
      </wfs:Keywords>
      <wfs:DefaultSRS>urn:ogc:def:crs:EPSG::4326</wfs:DefaultSRS>
      <wfs:OtherSRS>urn:ogc:def:crs:EPSG::3857</wfs:OtherSRS>
      <wfs:OutputFormats>
        <wfs:Format>application/gml+xml; version=3.2</wfs:Format>
        <wfs:Format>application/json</wfs:Format>
      </wfs:OutputFormats>
      <ows:WGS84BoundingBox>
        <ows:LowerCorner>-130 20</ows:LowerCorner>
        <ows:UpperCorner>-65 50</ows:UpperCorner>
      </ows:WGS84BoundingBox>
    </wfs:FeatureType>
  </wfs:FeatureTypeList>
</wfs:WFS_Capabilities>`

func TestParseCapabilities_OK(t *testing.T) {
	caps, err := wfs.ParseCapabilities(strings.NewReader(minimalCapsXML))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	if caps.Version != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0", caps.Version)
	}
	if caps.UpdateSequence != "42" {
		t.Errorf("UpdateSequence = %q", caps.UpdateSequence)
	}

	si := caps.ServiceIdentification
	if si.Title != "Test WFS" {
		t.Errorf("Title = %q", si.Title)
	}
	if got := len(si.Keywords); got != 2 {
		t.Errorf("Keywords: got %d, want 2", got)
	}
	if got := len(si.Versions); got != 2 || si.Versions[0] != "2.0.0" {
		t.Errorf("Versions = %+v", si.Versions)
	}

	if caps.ServiceProvider.ProviderName != "Acme" {
		t.Errorf("ProviderName = %q", caps.ServiceProvider.ProviderName)
	}

	ops := caps.OperationsMetadata.Operation
	if got := len(ops); got != 2 {
		t.Fatalf("Operations: got %d, want 2", got)
	}
	getFeat := findOp(ops, "GetFeature")
	if getFeat == nil {
		t.Fatalf("GetFeature operation missing")
	}
	if got := len(getFeat.DCP[0].HTTP.Post); got != 1 {
		t.Errorf("GetFeature POST endpoints: got %d, want 1", got)
	}

	fts := caps.FeatureTypeList.FeatureType
	if got := len(fts); got != 1 {
		t.Fatalf("FeatureTypes: got %d, want 1", got)
	}
	ft := fts[0]
	if ft.Name != "topp:states" {
		t.Errorf("FeatureType.Name = %q", ft.Name)
	}
	if ft.DefaultSRS != "urn:ogc:def:crs:EPSG::4326" {
		t.Errorf("DefaultSRS = %q", ft.DefaultSRS)
	}
	if got := len(ft.OutputFormats); got != 2 {
		t.Errorf("OutputFormats: got %d, want 2", got)
	}
	if ft.WGS84BoundingBox.LowerCorner != "-130 20" {
		t.Errorf("WGS84.LowerCorner = %q", ft.WGS84BoundingBox.LowerCorner)
	}
}

func findOp(ops []wfs.Operation, name string) *wfs.Operation {
	for i := range ops {
		if ops[i].Name == name {
			return &ops[i]
		}
	}
	return nil
}

func TestParseCapabilities_NilReader(t *testing.T) {
	if _, err := wfs.ParseCapabilities(nil); err == nil {
		t.Fatalf("expected error on nil reader")
	}
}

func TestParseCapabilities_Malformed(t *testing.T) {
	_, err := wfs.ParseCapabilities(strings.NewReader("<not-wfs/>"))
	if err == nil {
		t.Fatalf("expected error on non-WFS XML")
	}
	if !strings.Contains(err.Error(), "wfs: parse capabilities") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestGetCapabilities_GlobalDefaultVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wfs" {
			t.Errorf("path = %q, want /wfs", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("service") != "wfs" || q.Get("request") != "GetCapabilities" {
			t.Errorf("query = %+v", q)
		}
		if q.Get("version") != "2.0.0" {
			t.Errorf("version = %q, want 2.0.0 (default)", q.Get("version"))
		}
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	caps, err := c.WFS.GetCapabilities(context.Background(), wfs.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if caps.Version != "2.0.0" {
		t.Errorf("Version = %q", caps.Version)
	}
}

func TestGetCapabilities_WorkspaceScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topp/wfs" {
			t.Errorf("path = %q, want /topp/wfs", r.URL.Path)
		}
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	if _, err := c.WFS.InWorkspace("topp").GetCapabilities(context.Background(),
		wfs.GetCapabilitiesOptions{}); err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
}

func TestGetCapabilities_VersionOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("version") != "1.1.0" {
			t.Errorf("version = %q, want 1.1.0", r.URL.Query().Get("version"))
		}
		_, _ = io.WriteString(w, minimalCapsXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, _ = c.WFS.GetCapabilities(context.Background(), wfs.GetCapabilitiesOptions{
		Version: "1.1.0",
	})
}

func TestGetCapabilities_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such workspace", http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, err := c.WFS.InWorkspace("missing").GetCapabilities(context.Background(),
		wfs.GetCapabilitiesOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

const minimalSchemaXML = `<?xml version="1.0" encoding="UTF-8"?>
<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
            xmlns:gml="http://www.opengis.net/gml/3.2"
            xmlns:topp="http://www.openplans.org/topp"
            targetNamespace="http://www.openplans.org/topp"
            elementFormDefault="qualified">
  <xsd:import namespace="http://www.opengis.net/gml/3.2"
              schemaLocation="http://example.com/schemas/gml/3.2.1/gml.xsd"/>
  <xsd:complexType name="statesType">
    <xsd:complexContent>
      <xsd:extension base="gml:AbstractFeatureType">
        <xsd:sequence>
          <xsd:element minOccurs="0" name="the_geom" nillable="true" type="gml:MultiSurfacePropertyType"/>
          <xsd:element minOccurs="0" name="STATE_NAME" nillable="true" type="xsd:string"/>
          <xsd:element minOccurs="0" name="STATE_FIPS" nillable="true" type="xsd:string"/>
          <xsd:element minOccurs="0" name="PERSONS" nillable="true" type="xsd:double"/>
        </xsd:sequence>
      </xsd:extension>
    </xsd:complexContent>
  </xsd:complexType>
  <xsd:element name="states" substitutionGroup="gml:AbstractFeature" type="topp:statesType"/>
</xsd:schema>`

func TestParseFeatureSchema_OK(t *testing.T) {
	schema, err := wfs.ParseFeatureSchema(strings.NewReader(minimalSchemaXML))
	if err != nil {
		t.Fatalf("ParseFeatureSchema: %v", err)
	}
	if schema.TargetNamespace != "http://www.openplans.org/topp" {
		t.Errorf("TargetNamespace = %q", schema.TargetNamespace)
	}
	if got := len(schema.Imports); got != 1 {
		t.Errorf("Imports: got %d, want 1", got)
	}
	if got := len(schema.ComplexTypes); got != 1 {
		t.Fatalf("ComplexTypes: got %d, want 1", got)
	}
	if schema.ComplexTypes[0].Name != "statesType" {
		t.Errorf("ComplexType.Name = %q", schema.ComplexTypes[0].Name)
	}

	attrs := schema.Attributes("statesType")
	if got := len(attrs); got != 4 {
		t.Fatalf("Attributes(statesType): got %d, want 4", got)
	}
	if attrs[0].Name != "the_geom" || attrs[0].Type != "gml:MultiSurfacePropertyType" {
		t.Errorf("attrs[0] = %+v", attrs[0])
	}
	if !attrs[0].Nillable {
		t.Errorf("attrs[0].Nillable should be true")
	}
	if attrs[3].Type != "xsd:double" {
		t.Errorf("attrs[3].Type = %q", attrs[3].Type)
	}

	// Empty typeName picks the first complex type.
	if got := len(schema.Attributes("")); got != 4 {
		t.Errorf("Attributes(\"\"): got %d, want 4", got)
	}

	// Unknown typeName returns nil.
	if got := schema.Attributes("unknown"); got != nil {
		t.Errorf("Attributes(unknown) = %+v, want nil", got)
	}
}

func TestParseFeatureSchema_NilReader(t *testing.T) {
	if _, err := wfs.ParseFeatureSchema(nil); err == nil {
		t.Fatalf("expected error on nil reader")
	}
}

func TestParseFeatureSchema_Malformed(t *testing.T) {
	_, err := wfs.ParseFeatureSchema(strings.NewReader("<not-xsd/>"))
	if err == nil {
		t.Fatalf("expected error on non-XSD XML")
	}
	if !strings.Contains(err.Error(), "wfs: parse feature schema") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestDescribeFeatureType_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wfs" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("request") != "DescribeFeatureType" {
			t.Errorf("request = %q", q.Get("request"))
		}
		if q.Get("typeNames") != "topp:states" {
			t.Errorf("typeNames = %q", q.Get("typeNames"))
		}
		if q.Get("typeName") != "topp:states" {
			t.Errorf("typeName = %q (1.1.0 alias)", q.Get("typeName"))
		}
		_, _ = io.WriteString(w, minimalSchemaXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))

	schema, err := c.WFS.DescribeFeatureType(context.Background(),
		wfs.DescribeFeatureTypeOptions{TypeNames: []string{"topp:states"}})
	if err != nil {
		t.Fatalf("DescribeFeatureType: %v", err)
	}
	if got := len(schema.ComplexTypes); got != 1 {
		t.Errorf("ComplexTypes: got %d, want 1", got)
	}
	attrs := schema.Attributes("statesType")
	if got := len(attrs); got != 4 {
		t.Errorf("Attributes: got %d, want 4", got)
	}
}

func TestDescribeFeatureType_MultipleTypeNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("typeNames") != "topp:states,topp:counties" {
			t.Errorf("typeNames = %q", r.URL.Query().Get("typeNames"))
		}
		_, _ = io.WriteString(w, minimalSchemaXML)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, _ = c.WFS.DescribeFeatureType(context.Background(),
		wfs.DescribeFeatureTypeOptions{TypeNames: []string{"topp:states", "topp:counties"}})
}

func TestDescribeFeatureType_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such type", http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := geoserver.New(srv.URL, geoserver.WithBasicAuth("u", "p"))
	_, err := c.WFS.DescribeFeatureType(context.Background(),
		wfs.DescribeFeatureTypeOptions{TypeNames: []string{"topp:missing"}})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestClient_IsGlobal(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	if !c.WFS.IsGlobal() {
		t.Errorf("freshly constructed WFS client should be global")
	}
	scoped := c.WFS.InWorkspace("topp")
	if scoped.IsGlobal() {
		t.Errorf("workspace-scoped client reports IsGlobal=true")
	}
	if scoped.Workspace() != "topp" {
		t.Errorf("Workspace() = %q", scoped.Workspace())
	}
	if !c.WFS.IsGlobal() {
		t.Errorf("InWorkspace mutated original")
	}
}
