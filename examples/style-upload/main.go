// style-upload is a runnable example: stream an SLD file from disk into
// GeoServer using UploadStyle. The example uses one of the SLDs shipped
// under testdata/ for portability.
//
// Run with:
//
//	go run ./examples/style-upload
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hishamkaram/geoserver"
)

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	gs := geoserver.New(url, user, pass, geoserver.WithTimeout(30*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		ws        = "examples_styleupload_demo"
		styleName = "examples_airports_sld"
	)
	sldPath := envOr("STYLE_FILE", filepath.Join("testdata", "airports.sld"))

	_, _ = gs.DeleteWorkspaceContext(ctx, ws, true)
	if _, err := gs.CreateWorkspaceContext(ctx, ws); err != nil {
		fatal("create workspace: %v", err)
	}
	defer func() { _, _ = gs.DeleteWorkspaceContext(ctx, ws, true) }()

	f, err := os.Open(sldPath)
	if err != nil {
		fatal("open SLD %q: %v", sldPath, err)
	}
	defer func() { _ = f.Close() }()

	// UploadStyle streams the io.Reader directly to GeoServer — no full
	// in-memory slurp. Pass overwrite=true to replace any existing style
	// with the same name (will create-then-PUT under the hood).
	uploaded, err := gs.UploadStyleContext(ctx, f, ws, styleName, true)
	if err != nil {
		fatal("upload style: %v", err)
	}
	fmt.Printf("uploaded SLD %q to workspace %q (uploaded=%t)\n", styleName, ws, uploaded)

	// Fetch the style metadata back as a sanity check.
	style, err := gs.GetStyleContext(ctx, ws, styleName)
	if err != nil {
		fatal("fetch style: %v", err)
	}
	fmt.Printf("style: name=%q format=%q filename=%q\n", style.Name, style.Format, style.Filename)
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
