//go:build integration

package wms_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/ows/wms"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestWMS_GetCapabilities_Global_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	caps, err := c.WMS.GetCapabilities(ctx, wms.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities (global): %v", err)
	}
	if caps.Version == "" {
		t.Errorf("Version is empty: %+v", caps)
	}
	if caps.Service.Name == "" {
		t.Errorf("Service.Name is empty: %+v", caps.Service)
	}
}

func TestWMS_GetCapabilities_Workspace_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	// A fresh workspace with no layers still has a valid (empty)
	// capabilities document.
	caps, err := c.WMS.InWorkspace(wsName).GetCapabilities(ctx, wms.GetCapabilitiesOptions{})
	if err != nil {
		t.Fatalf("GetCapabilities (workspace): %v", err)
	}
	if caps.Version == "" {
		t.Errorf("Version is empty for workspace-scoped caps")
	}
}

func TestWMS_GetCapabilities_NonExistentWorkspace_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// GeoServer's wire behavior on a missing workspace: some versions
	// return 404, others return a service exception with HTTP 200 and
	// an XML error body that fails to parse as Capabilities. Either
	// surfaces as an error; we accept both.
	_, err := c.WMS.InWorkspace("definitely-not-a-real-workspace-zz").
		GetCapabilities(ctx, wms.GetCapabilitiesOptions{})
	if err == nil {
		t.Fatalf("expected error for missing workspace")
	}
	// If it's a typed APIError, the sentinel chain works. If it's a
	// parse error from a service-exception body returned with 200,
	// it'll be a wrapped xml-decode error — that's still a non-nil
	// error and meets the assertion.
	if errors.Is(err, geoserver.ErrNotFound) {
		// strict path
		return
	}
	// Otherwise just ensure we got a usable error message.
	if err.Error() == "" {
		t.Fatalf("error has empty message: %v", err)
	}
}
