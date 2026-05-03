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
