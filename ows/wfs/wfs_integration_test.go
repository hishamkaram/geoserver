//go:build integration

package wfs_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/ows/wfs"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestWFS_GetCapabilities_Global_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	caps, err := c.WFS.GetCapabilities(ctx, wfs.GetCapabilitiesOptions{})
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

func TestWFS_GetCapabilities_Workspace_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	caps, err := c.WFS.InWorkspace(wsName).GetCapabilities(ctx, wfs.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities (workspace): %v", err)
	}
	if caps.Version == "" {
		t.Errorf("Version is empty for workspace-scoped caps")
	}
}

func TestWFS_DescribeFeatureType_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Use one of GeoServer's default-install feature types
	// (sf:archsites is shipped by default with the workshop data).
	schema, err := c.WFS.DescribeFeatureType(ctx, wfs.DescribeFeatureTypeOptions{
		TypeNames: []string{"sf:archsites"},
	})
	if err != nil {
		t.Fatalf("DescribeFeatureType: %v", err)
	}
	if schema.TargetNamespace == "" {
		t.Errorf("TargetNamespace is empty: %+v", schema)
	}
	if got := len(schema.ComplexTypes); got == 0 {
		t.Fatalf("ComplexTypes empty — schema didn't decode")
	}
	attrs := schema.Attributes("")
	if len(attrs) == 0 {
		t.Errorf("Attributes empty — sf:archsites should have at least geometry + cat + str1")
	}
}

func TestWFS_GetCapabilities_Version110_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Explicitly request 1.1.0 — the type tree handles both 1.1.0
	// and 2.0.0 since the root element name is identical.
	caps, err := c.WFS.GetCapabilities(ctx, wfs.GetCapabilitiesOptions{Version: "1.1.0"})
	if err != nil {
		t.Fatalf("GetCapabilities v1.1.0: %v", err)
	}
	if caps.Version == "" {
		t.Errorf("Version is empty: %+v", caps)
	}
}
