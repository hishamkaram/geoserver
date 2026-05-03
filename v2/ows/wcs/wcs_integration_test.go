//go:build integration

package wcs_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/ows/wcs"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestWCS_GetCapabilities_Global_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	caps, err := c.WCS.GetCapabilities(ctx, wcs.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities (global): %v", err)
	}
	if caps.Version == "" {
		t.Errorf("Version is empty: %+v", caps)
	}
	if caps.ServiceIdentification.ServiceType == "" {
		t.Errorf("ServiceType is empty: %+v", caps.ServiceIdentification)
	}
}

func TestWCS_DescribeCoverage_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Discover an actual coverage by listing capabilities first —
	// avoids hard-coding a coverage ID that varies across data
	// directory variants.
	caps, err := c.WCS.GetCapabilities(ctx, wcs.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if len(caps.Contents.CoverageSummary) == 0 {
		t.Skip("no published coverages — skipping DescribeCoverage")
	}
	id := caps.Contents.CoverageSummary[0].CoverageID

	descs, err := c.WCS.DescribeCoverage(ctx, wcs.DescribeCoverageOptions{
		CoverageIDs: []string{id},
	})
	if err != nil {
		t.Fatalf("DescribeCoverage(%q): %v", id, err)
	}
	if got := len(descs.CoverageDescription); got != 1 {
		t.Fatalf("CoverageDescription: got %d, want 1", got)
	}
	d := descs.CoverageDescription[0]
	if d.CoverageID != id {
		t.Errorf("CoverageID = %q, want %q", d.CoverageID, id)
	}
	if d.BoundedBy.Envelope.LowerCorner == "" {
		t.Errorf("Envelope.LowerCorner is empty: %+v", d.BoundedBy)
	}
	if got := len(d.RangeType.DataRecord.Field); got == 0 {
		t.Errorf("RangeType.Fields empty — coverage with no bands shouldn't happen")
	}
}

func TestWCS_GetCapabilities_Workspace_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	caps, err := c.WCS.InWorkspace(wsName).GetCapabilities(ctx, wcs.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities (workspace): %v", err)
	}
	if caps.Version == "" {
		t.Errorf("Version is empty for workspace-scoped caps")
	}
}
