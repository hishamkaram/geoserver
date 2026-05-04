package gwc_test

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/gwc"
)

// ===== Global =====

func TestGlobal_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gwc/rest/global.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"global":{"identifier":"gwc","runtimeStatsEnabled":true,"backendTimeout":120,"wmtsCiteCompliant":false,"location":"gwc/geowebcache.xml","version":"1.8.0"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	g, err := c.GWC.Global().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g.Identifier != "gwc" || !g.RuntimeStatsEnabled || g.BackendTimeout != 120 || g.WMTSCiteCompliant {
		t.Errorf("global = %+v", g)
	}
}

func TestGlobal_Update_BodyEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/gwc/rest/global.json" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"global":{`) || !strings.Contains(s, `"backendTimeout":300`) {
			t.Errorf("missing envelope or fields; got %s", s)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GWC.Global().Update(context.Background(), &gwc.Global{
		RuntimeStatsEnabled: true,
		WMTSCiteCompliant:   false,
		BackendTimeout:      300,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestGlobal_Update_NilRejected(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	})))
	if err := c.GWC.Global().Update(context.Background(), nil); err == nil {
		t.Fatal("expected nil-config error")
	}
}

// ===== Gridsets =====

func TestGridsets_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gwc/rest/gridsets.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `["EPSG:4326","WebMercatorQuad","UTM50WGS84Quad"]`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GWC.Gridsets().List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 || got[0] != "EPSG:4326" {
		t.Errorf("List = %+v", got)
	}
}

func TestGridsets_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Colon may or may not be percent-encoded.
		if !strings.HasPrefix(r.URL.Path, "/gwc/rest/gridsets/") || !strings.HasSuffix(r.URL.Path, ".json") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"gridSet":{"name":"EPSG:4326","srs":{"number":4326},"extent":{"coords":[-180,-90,180,90]},"yCoordinateFirst":true,"metersPerUnit":111319.49,"scaleNames":["EPSG:4326:0","EPSG:4326:1"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	g, err := c.GWC.Gridsets().Get(context.Background(), "EPSG:4326")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g.Name != "EPSG:4326" || g.SRS.Number != 4326 || !g.YCoordinateFirst {
		t.Errorf("gridSet = %+v", g)
	}
	if len(g.Extent.Coords) != 4 || g.Extent.Coords[0] != -180 {
		t.Errorf("extent = %+v", g.Extent)
	}
}

func TestGridsets_Get_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GWC.Gridsets().Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGridsets_Get_EmptyName(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})))
	if _, err := c.GWC.Gridsets().Get(context.Background(), ""); err == nil {
		t.Fatal("expected empty-name error")
	}
}

func TestGridsets_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.GWC.Gridsets().Delete(context.Background(), "custom-set"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// ===== MassTruncate =====

func TestMassTruncate_Capabilities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gwc/rest/masstruncate.xml" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<massTruncateRequests><requestType>truncateLayer</requestType><requestType>truncateParameters</requestType><requestType>truncateOrphans</requestType><requestType>truncateExtent</requestType></massTruncateRequests>`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	caps, err := c.GWC.MassTruncate().Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if len(caps) != 4 || caps[0] != gwc.TruncateLayer {
		t.Errorf("caps = %v", caps)
	}
}

func TestMassTruncate_TruncateLayer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/gwc/rest/masstruncate" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "text/xml" {
			t.Errorf("Content-Type = %q, want text/xml", got)
		}
		body, _ := io.ReadAll(r.Body)
		var got gwc.MassTruncateLayerRequest
		if err := xml.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode: %v; body=%s", err, body)
		}
		if got.LayerName != "topp:states" {
			t.Errorf("layerName = %q", got.LayerName)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.GWC.MassTruncate().TruncateLayer(context.Background(), "topp:states"); err != nil {
		t.Fatalf("TruncateLayer: %v", err)
	}
}

func TestMassTruncate_TruncateOrphans(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "<truncateOrphans>") {
			t.Errorf("body missing <truncateOrphans>; got %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.GWC.MassTruncate().TruncateOrphans(context.Background()); err != nil {
		t.Fatalf("TruncateOrphans: %v", err)
	}
}

func TestMassTruncate_RejectsEmpty(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	})))
	if err := c.GWC.MassTruncate().TruncateLayer(context.Background(), ""); err == nil {
		t.Error("expected empty-layerName error")
	}
	if err := c.GWC.MassTruncate().TruncateExtent(context.Background(), nil); err == nil {
		t.Error("expected nil-request error")
	}
	if err := c.GWC.MassTruncate().TruncateExtent(context.Background(), &gwc.MassTruncateExtentRequest{LayerName: "x"}); err == nil {
		t.Error("expected nil-Bounds error")
	}
}
