// style-upload is a runnable v2 example: register a style's metadata,
// then upload an SLD body via UploadSLD. Demonstrates the two-step
// publish flow and the workspace-scoped Accept-quirk handling.
//
//	go run ./v2/examples/style-upload
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/styles"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// inlineSLD is a tiny SLD that paints polygons solid red. Embedded so the
// example doesn't need an external file; in real code you'd typically
// open a file or fetch from S3 / a config bucket.
const inlineSLD = `<?xml version="1.0" encoding="UTF-8"?>
<StyledLayerDescriptor version="1.0.0"
    xsi:schemaLocation="http://www.opengis.net/sld StyledLayerDescriptor.xsd"
    xmlns="http://www.opengis.net/sld"
    xmlns:ogc="http://www.opengis.net/ogc"
    xmlns:xlink="http://www.w3.org/1999/xlink"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <NamedLayer>
    <Name>red_polygon</Name>
    <UserStyle>
      <Title>Red polygon</Title>
      <FeatureTypeStyle>
        <Rule>
          <PolygonSymbolizer>
            <Fill>
              <CssParameter name="fill">#FF0000</CssParameter>
            </Fill>
          </PolygonSymbolizer>
        </Rule>
      </FeatureTypeStyle>
    </UserStyle>
  </NamedLayer>
</StyledLayerDescriptor>`

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	c, err := geoserver.New(url,
		geoserver.WithBasicAuth(user, pass),
		geoserver.WithTimeout(30*time.Second),
		geoserver.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))),
	)
	if err != nil {
		fatal("construct client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		workspace = "v2_examples_styles"
		styleName = "red_polygon"
	)

	// Best-effort cleanup from a prior run.
	_ = c.Workspaces.Delete(ctx, workspace, workspaces.DeleteOptions{Recurse: true})

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: workspace}); err != nil {
		fatal("create workspace: %v", err)
	}
	fmt.Printf("created workspace %q\n", workspace)

	ws := c.Styles.InWorkspace(workspace)

	// Step 1: register the style metadata. Workspace-scoped POST
	// automatically uses Accept: */* under the hood (the Accept-quirk
	// workaround is internal to v2).
	if err := ws.Create(ctx, &styles.Style{Name: styleName}); err != nil {
		fatal("create style metadata: %v", err)
	}
	fmt.Printf("registered style metadata %q\n", styleName)

	// Step 2: upload the SLD body. Default Content-Type is
	// application/vnd.ogc.sld+xml; override via UploadOptions.Format
	// for SE 1.1 / GeoCSS / etc.
	if err := ws.UploadSLD(ctx, styleName, bytes.NewReader([]byte(inlineSLD)), styles.UploadOptions{}); err != nil {
		fatal("upload SLD: %v", err)
	}
	fmt.Printf("uploaded SLD body for %q (%d bytes)\n", styleName, len(inlineSLD))

	// Read back the metadata.
	got, err := ws.Get(ctx, styleName)
	if err != nil {
		fatal("fetch style: %v", err)
	}
	fmt.Printf("style: name=%q format=%q filename=%q\n", got.Name, got.Format, got.Filename)

	// Cleanup.
	if err := ws.Delete(ctx, styleName, styles.DeleteOptions{Purge: true}); err != nil {
		fmt.Fprintf(os.Stderr, "warn: delete style: %v\n", err)
	}
	if err := c.Workspaces.Delete(ctx, workspace, workspaces.DeleteOptions{Recurse: true}); err != nil {
		fmt.Fprintf(os.Stderr, "warn: cleanup workspace: %v\n", err)
	}

	// Verify the style is gone.
	_, err = ws.Get(ctx, styleName)
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("verified style and workspace cleaned up")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
