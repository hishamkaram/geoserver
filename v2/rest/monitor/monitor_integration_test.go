//go:build integration

package monitor_test

import (
	"io"
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/monitor"
)

// TestMonitor_List_Integration drives a non-empty audit log: every
// REST call we make against GeoServer is itself recorded by the
// monitor, so by the time we ask for the list there is at least one
// entry — including possibly this very call.
func TestMonitor_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Trigger at least one request so the audit log has data even
	// on a freshly started stack. About.Version is cheap.
	if _, err := c.About.Version(ctx); err != nil {
		t.Fatalf("warm-up About.Version: %v", err)
	}

	got, err := c.Monitor.List(ctx, monitor.ListOptions{Count: 5})
	if err != nil {
		t.Fatalf("Monitor.List: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one audit-log entry, got empty list")
	}
	for i, r := range got {
		if r.ID == 0 {
			t.Errorf("row[%d] has zero ID: %+v", i, r)
		}
	}
}

// TestMonitor_ListRaw_Integration verifies the streaming path works
// — useful for callers integrating with reporting pipelines.
func TestMonitor_ListRaw_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	rc, err := c.Monitor.ListRaw(ctx, monitor.ListOptions{Count: 1})
	if err != nil {
		t.Fatalf("Monitor.ListRaw: %v", err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("empty CSV body")
	}
}

// TestMonitor_Get_Integration fetches a known-existing audit row by
// ID. It first lists to find a real ID rather than guessing.
func TestMonitor_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	if _, err := c.About.Version(ctx); err != nil {
		t.Fatalf("warm-up: %v", err)
	}

	rows, err := c.Monitor.List(ctx, monitor.ListOptions{Count: 1, Order: "Id;DESC"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) == 0 {
		t.Skip("no audit entries available; skipping Get round-trip")
	}
	id := rows[0].ID
	got, err := c.Monitor.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get(%d): %v", id, err)
	}
	if got.ID != id {
		t.Errorf("got.ID = %d, want %d", got.ID, id)
	}
}

// TestMonitor_List_FieldsFilter exercises the fields query parameter
// — a stricter projection than "all fields". Upstream parser only
// accepts a single column name today (see [monitor.ListOptions.Fields]
// godoc); this test honors that constraint.
func TestMonitor_List_FieldsFilter_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	if _, err := c.About.Version(ctx); err != nil {
		t.Fatalf("warm-up: %v", err)
	}

	rows, err := c.Monitor.List(ctx, monitor.ListOptions{
		Fields: []string{"Id"},
		Count:  3,
	})
	if err != nil {
		t.Fatalf("List with fields: %v", err)
	}
	for _, r := range rows {
		if r.ID == 0 {
			t.Errorf("expected non-zero ID with field projection: %+v", r)
		}
	}
}
