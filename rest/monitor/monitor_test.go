package monitor_test

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
	"github.com/hishamkaram/geoserver/v2/rest/monitor"
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

const sampleCSV = `Id,Path,Service,Operation,HttpMethod,QueryString,Category,StartTime,EndTime,TotalTime,Status,ResponseStatus,ResponseLength,RemoteAddr,RemoteUser,Resources
1,/rest/about/version.json,,,GET,,REST,2026-04-23T16:16:44.000,2026-04-23T16:16:44.025,25,FINISHED,200,300,127.0.0.1,admin,
2,/topp/wms,WMS,GetMap,GET,service=WMS,OWS,2026-04-23T16:17:01.500,2026-04-23T16:17:01.700,200,FINISHED,200,12345,10.0.0.5,,"[topp:states]"
`

func TestList_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/monitor/requests.csv" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/csv")
		_, _ = io.WriteString(w, sampleCSV)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Monitor.List(context.Background(), monitor.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	r0 := got[0]
	if r0.ID != 1 || r0.Path != "/rest/about/version.json" || r0.Status != "FINISHED" || r0.ResponseStatus != 200 || r0.RemoteUser != "admin" {
		t.Errorf("row[0] = %+v", r0)
	}
	if r0.StartTime.IsZero() || r0.EndTime.IsZero() {
		t.Errorf("row[0] times zero: %+v", r0)
	}
	r1 := got[1]
	if r1.Service != "WMS" || r1.Operation != "GetMap" || r1.TotalTime != 200 || len(r1.Resources) != 1 || r1.Resources[0] != "topp:states" {
		t.Errorf("row[1] = %+v", r1)
	}
}

func TestList_QueryParameters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("from") != "2026-04-23" {
			t.Errorf("from = %q", q.Get("from"))
		}
		if q.Get("to") != "2026-04-24" {
			t.Errorf("to = %q", q.Get("to"))
		}
		if q.Get("filter") != "service:EQ:WMS" {
			t.Errorf("filter = %q", q.Get("filter"))
		}
		if q.Get("count") != "10" {
			t.Errorf("count = %q", q.Get("count"))
		}
		if q.Get("fields") != "Id,Path" {
			t.Errorf("fields = %q", q.Get("fields"))
		}
		if q.Get("live") != "false" {
			t.Errorf("live = %q", q.Get("live"))
		}
		_, _ = io.WriteString(w, "Id,Path\n")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	live := false
	_, err := c.Monitor.List(context.Background(), monitor.ListOptions{
		From: "2026-04-23", To: "2026-04-24",
		Filter: "service:EQ:WMS",
		Count:  10,
		Fields: []string{"Id", "Path"},
		Live:   &live,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/monitor/requests/42.csv" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, "Id,Path\n42,/rest/test\n")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Monitor.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != 42 || got.Path != "/rest/test" {
		t.Errorf("got = %+v", got)
	}
}

func TestGet_RejectsNonPositive(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	})))
	if _, err := c.Monitor.Get(context.Background(), 0); err == nil {
		t.Error("expected error for id=0")
	}
	if _, err := c.Monitor.Get(context.Background(), -1); err == nil {
		t.Error("expected error for id=-1")
	}
}

func TestList_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Monitor.List(context.Background(), monitor.ListOptions{})
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound (extension absent), got %v", err)
	}
}

func TestListRaw_StreamPassThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, sampleCSV)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rc, err := c.Monitor.ListRaw(context.Background(), monitor.ListOptions{})
	if err != nil {
		t.Fatalf("ListRaw: %v", err)
	}
	defer func() { _ = rc.Close() }()
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(string(body), "Id,Path") {
		t.Errorf("body did not start with header; got %q", body[:50])
	}
}
