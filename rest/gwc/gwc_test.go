package gwc_test

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
	"github.com/hishamkaram/geoserver/v2/rest/gwc"
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

// ===== Layers =====

func TestLayers_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gwc/rest/layers.json" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `["topp:states","sf:archsites","ne:countries"]`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GWC.Layers().List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 || got[0] != "topp:states" {
		t.Errorf("List = %+v", got)
	}
}

func TestLayers_Get_XML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL builder percent-encodes the colon in qualified names.
		if r.URL.Path != "/gwc/rest/layers/topp%3Astates.xml" && r.URL.Path != "/gwc/rest/layers/topp:states.xml" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<GeoServerLayer>
  <id>LayerInfoImpl-x</id>
  <enabled>true</enabled>
  <inMemoryCached>true</inMemoryCached>
  <name>topp:states</name>
  <mimeFormats>
    <string>image/png</string>
    <string>image/jpeg</string>
  </mimeFormats>
  <gridSubsets>
    <gridSubset><gridSetName>EPSG:4326</gridSetName></gridSubset>
    <gridSubset><gridSetName>EPSG:900913</gridSetName></gridSubset>
  </gridSubsets>
  <metaWidthHeight><int>4</int><int>4</int></metaWidthHeight>
  <expireCache>0</expireCache>
  <gutter>0</gutter>
</GeoServerLayer>`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GWC.Layers().Get(context.Background(), "topp:states")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "topp:states" || !got.Enabled {
		t.Errorf("Get = %+v", got)
	}
	if got.MimeFormats == nil || len(got.MimeFormats.String) != 2 {
		t.Errorf("MimeFormats = %+v", got.MimeFormats)
	}
	if got.GridSubsets == nil || len(got.GridSubsets.GridSubset) != 2 {
		t.Errorf("GridSubsets = %+v", got.GridSubsets)
	}
	if got.MetaWidthHeight == nil || len(got.MetaWidthHeight.Int) != 2 {
		t.Errorf("MetaWidthHeight = %+v", got.MetaWidthHeight)
	}
}

func TestLayers_Put_XMLBody(t *testing.T) {
	var captured struct {
		Method, Path, ContentType string
		Body                      []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		captured.Body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GWC.Layers().Put(context.Background(), "topp:states", &gwc.LayerConfig{
		Name:        "topp:states",
		Enabled:     true,
		MimeFormats: &gwc.MimeFormats{String: []string{"image/png"}},
	})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if captured.Method != http.MethodPut {
		t.Errorf("Method = %q", captured.Method)
	}
	if captured.ContentType != "application/xml" {
		t.Errorf("Content-Type = %q", captured.ContentType)
	}
	if !strings.HasPrefix(string(captured.Body), "<GeoServerLayer>") {
		t.Errorf("body not XML-shaped: %q", string(captured.Body))
	}
	if !strings.Contains(string(captured.Body), "<name>topp:states</name>") {
		t.Errorf("body missing name: %q", string(captured.Body))
	}
}

func TestLayers_Delete(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %q", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.GWC.Layers().Delete(context.Background(), "topp:states"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !called {
		t.Errorf("server not called")
	}
}

func TestLayers_Validation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	if _, err := c.GWC.Layers().Get(context.Background(), ""); err == nil {
		t.Errorf("Get empty name: expected error")
	}
	if err := c.GWC.Layers().Put(context.Background(), "", &gwc.LayerConfig{}); err == nil {
		t.Errorf("Put empty name: expected error")
	}
	if err := c.GWC.Layers().Put(context.Background(), "x", nil); err == nil {
		t.Errorf("Put nil config: expected error")
	}
	if err := c.GWC.Layers().Delete(context.Background(), ""); err == nil {
		t.Errorf("Delete empty name: expected error")
	}
}

// ===== Seed =====

func TestSeed_Submit_BodyShape(t *testing.T) {
	var capturedBody json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q", r.Method)
		}
		// URL builder percent-encodes the colon.
		if !strings.Contains(r.URL.Path, "topp") || !strings.HasSuffix(r.URL.Path, ".json") {
			t.Errorf("Path = %q", r.URL.Path)
		}
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GWC.Seed().Submit(context.Background(), "topp:states", &gwc.SeedRequest{
		SRS:         gwc.SRS{Number: 4326},
		ZoomStart:   0,
		ZoomStop:    5,
		Format:      "image/png",
		Type:        gwc.OpTruncate,
		ThreadCount: 1,
		GridSetID:   "EPSG:4326",
		Bounds: &gwc.Bounds{Coords: gwc.BoundsCoords{
			Double: []float64{-180, -90, 180, 90},
		}},
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	body := string(capturedBody)
	if !strings.HasPrefix(body, `{"seedRequest":`) {
		t.Errorf("body envelope wrong: %q", body)
	}
	if !strings.Contains(body, `"name":"topp:states"`) {
		t.Errorf("body missing name (auto-fill from layer arg): %q", body)
	}
	if !strings.Contains(body, `"type":"truncate"`) {
		t.Errorf("body missing type: %q", body)
	}
	if !strings.Contains(body, `"srs":{"number":4326}`) {
		t.Errorf("body missing srs: %q", body)
	}
	if !strings.Contains(body, `"double":[-180,-90,180,90]`) {
		t.Errorf("body missing bounds: %q", body)
	}
}

func TestSeed_Submit_Validation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))

	if err := c.GWC.Seed().Submit(context.Background(), "", &gwc.SeedRequest{Type: gwc.OpSeed}); err == nil {
		t.Errorf("Submit empty layer: expected error")
	}
	if err := c.GWC.Seed().Submit(context.Background(), "topp:states", nil); err == nil {
		t.Errorf("Submit nil request: expected error")
	}
	if err := c.GWC.Seed().Submit(context.Background(), "topp:states", &gwc.SeedRequest{}); err == nil {
		t.Errorf("Submit empty Type: expected error")
	}
}

func TestSeed_Status_LongArrayArrayUnmarshal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"long-array-array":[
			[100, 1000, 30, 1, 1],
			[2000, 5000, 0, 2, 2]
		]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GWC.Seed().StatusAll(context.Background())
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(got.Tasks) != 2 {
		t.Fatalf("Tasks = %+v", got.Tasks)
	}
	if got.Tasks[0].TaskID != 1 || got.Tasks[0].Status != gwc.StatusRunning {
		t.Errorf("Tasks[0] = %+v", got.Tasks[0])
	}
	if got.Tasks[1].Status != gwc.StatusDone {
		t.Errorf("Tasks[1].Status = %v", got.Tasks[1].Status)
	}
}

func TestSeed_StatusAll_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"long-array-array":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GWC.Seed().StatusAll(context.Background())
	if err != nil {
		t.Fatalf("StatusAll: %v", err)
	}
	if len(got.Tasks) != 0 {
		t.Errorf("expected empty Tasks, got %+v", got.Tasks)
	}
}

func TestSeed_TaskStatus_String(t *testing.T) {
	cases := []struct {
		s    gwc.SeedTaskStatus
		want string
	}{
		{gwc.StatusAborted, "ABORTED"},
		{gwc.StatusPending, "PENDING"},
		{gwc.StatusRunning, "RUNNING"},
		{gwc.StatusDone, "DONE"},
		{gwc.SeedTaskStatus(99), "UNKNOWN(99)"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("Status(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

func TestSeed_KillAll(t *testing.T) {
	var captured struct {
		Method, Path, ContentType, Body string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		captured.Body = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.GWC.Seed().KillAll(context.Background()); err != nil {
		t.Fatalf("KillAll: %v", err)
	}
	if captured.Path != "/gwc/rest/seed" {
		t.Errorf("Path = %q", captured.Path)
	}
	if captured.ContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", captured.ContentType)
	}
	if captured.Body != "kill_all=all" {
		t.Errorf("Body = %q", captured.Body)
	}
}

// ===== DiskQuota =====

func TestDiskQuota_Get_ClassNameEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gwc/rest/diskquota.json" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"org.geowebcache.diskquota.DiskQuotaConfig":{
			"enabled":true,
			"cacheCleanUpFrequency":10,
			"cacheCleanUpUnits":"MINUTES",
			"globalExpirationPolicyName":"LRU",
			"globalQuota":{"id":0,"bytes":1073741824}
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GWC.DiskQuota().Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Enabled || got.GlobalExpirationPolicyName != "LRU" {
		t.Errorf("Get = %+v", got)
	}
	if got.GlobalQuota == nil || got.GlobalQuota.Bytes != 1073741824 {
		t.Errorf("GlobalQuota = %+v", got.GlobalQuota)
	}
}

func TestDiskQuota_Update_XMLBody(t *testing.T) {
	var captured struct {
		Method, Path, ContentType string
		Body                      []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		captured.Body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GWC.DiskQuota().Update(context.Background(), &gwc.DiskQuota{
		Enabled:                    true,
		GlobalExpirationPolicyName: "LFU",
		GlobalQuota:                &gwc.Quota{Bytes: 2 * 1024 * 1024 * 1024},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if captured.Method != http.MethodPut {
		t.Errorf("Method = %q", captured.Method)
	}
	if captured.Path != "/gwc/rest/diskquota.xml" {
		t.Errorf("Path = %q (want .xml suffix — JSON PUT is rejected by GWC)", captured.Path)
	}
	if captured.ContentType != "application/xml" {
		t.Errorf("Content-Type = %q (must be XML, GWC's PUT parser rejects JSON)", captured.ContentType)
	}
	body := string(captured.Body)
	if !strings.HasPrefix(body, `<org.geowebcache.diskquota.DiskQuotaConfig>`) {
		t.Errorf("body root wrong: %q", body)
	}
	if !strings.Contains(body, `<globalExpirationPolicyName>LFU</globalExpirationPolicyName>`) {
		t.Errorf("body missing policy: %q", body)
	}
	// Wire-quirk assertion: must use value/units, NOT bytes.
	if !strings.Contains(body, `<value>2147483648</value>`) {
		t.Errorf("body missing <value>: %q", body)
	}
	if !strings.Contains(body, `<units>B</units>`) {
		t.Errorf("body missing <units>: %q", body)
	}
	if strings.Contains(body, `<bytes>`) {
		t.Errorf("body has <bytes> (forbidden by GWC PUT parser): %q", body)
	}
}

func TestDiskQuota_Update_Validation(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	if err := c.GWC.DiskQuota().Update(context.Background(), nil); err == nil {
		t.Errorf("Update nil: expected error")
	}
}

// ===== Cross-cutting =====

func TestErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such layer", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GWC.Layers().Get(context.Background(), "missing:layer")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
