//go:build integration

package styles_test

import (
	"errors"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/styles"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

const testSLD = `<?xml version="1.0" encoding="UTF-8"?>
<StyledLayerDescriptor version="1.0.0"
    xsi:schemaLocation="http://www.opengis.net/sld StyledLayerDescriptor.xsd"
    xmlns="http://www.opengis.net/sld"
    xmlns:ogc="http://www.opengis.net/ogc"
    xmlns:xlink="http://www.w3.org/1999/xlink"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <NamedLayer>
    <Name>integration_test_polygon</Name>
    <UserStyle>
      <Title>Integration test polygon</Title>
      <FeatureTypeStyle>
        <Rule>
          <PolygonSymbolizer>
            <Fill>
              <CssParameter name="fill">#00FF00</CssParameter>
            </Fill>
          </PolygonSymbolizer>
        </Rule>
      </FeatureTypeStyle>
    </UserStyle>
  </NamedLayer>
</StyledLayerDescriptor>`

func TestStyles_Workspace_TwoStepPublish_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	styleName := testenv.UniqueName(t, "style")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	ws := c.Styles.InWorkspace(wsName)

	// Empty-collection wire path: workspace-scoped /styles is initially empty.
	empty, err := ws.List(ctx, styles.ListOptions{})
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty list, got %+v", empty)
	}

	// Step 1: create style metadata. The workspace-scoped POST exercises
	// the Accept: */* quirk — should not return 500.
	if err := ws.Create(ctx, &styles.Style{Name: styleName}); err != nil {
		t.Fatalf("Create style metadata: %v", err)
	}

	// Step 2: upload SLD body. Default content-type is application/vnd.ogc.sld+xml.
	if err := ws.UploadSLD(ctx, styleName, strings.NewReader(testSLD), styles.UploadOptions{}); err != nil {
		t.Fatalf("UploadSLD: %v", err)
	}

	// Read back metadata.
	got, err := ws.Get(ctx, styleName)
	if err != nil {
		t.Fatalf("Get style: %v", err)
	}
	if got.Name != styleName {
		t.Fatalf("Style.Name = %q", got.Name)
	}
	if got.Filename == "" {
		t.Errorf("Filename should be auto-derived: %+v", got)
	}

	// List should now include the style.
	all, err := ws.List(ctx, styles.ListOptions{})
	if err != nil {
		t.Fatalf("List populated: %v", err)
	}
	found := false
	for _, s := range all {
		if s.Name == styleName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("style %q not found in List: %+v", styleName, all)
	}

	// Delete with purge — also removes the on-disk SLD file.
	if err := ws.Delete(ctx, styleName, styles.DeleteOptions{Purge: true}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := ws.Get(ctx, styleName); !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestStyles_Global_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Default GeoServer ships with built-in global styles (point, line,
	// polygon, raster, …). The list must succeed and produce a non-empty
	// slice.
	got, err := c.Styles.List(ctx, styles.ListOptions{})
	if err != nil {
		t.Fatalf("global List: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected built-in styles in default install, got empty")
	}
}
