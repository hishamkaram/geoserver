// error-handling is a runnable example: match GeoServer errors via
// errors.Is(err, geoserver.ErrNotFound) and inspect the typed
// *geoserver.Error via errors.As. Demonstrates the v1.1 typed-error model.
//
// Run with:
//
//	go run ./examples/error-handling
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hishamkaram/geoserver"
)

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	gs := geoserver.New(url, user, pass, geoserver.WithTimeout(10*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Sentinel matching via errors.Is. This is the recommended pattern —
	//    works regardless of message format and survives wrap chains.
	_, err := gs.GetWorkspaceContext(ctx, "definitely_not_a_real_workspace")
	switch {
	case err == nil:
		fmt.Println("unexpected: workspace existed")
	case errors.Is(err, geoserver.ErrNotFound):
		fmt.Println("404 mapped to ErrNotFound — handle as caller chooses")
	case errors.Is(err, geoserver.ErrUnauthorized):
		fmt.Println("401 — bad credentials")
	default:
		fmt.Printf("other error: %v\n", err)
	}

	// 2. Type assertion via errors.As when you need StatusCode / Body /
	//    Op / URL. Useful for logging or retry decisions.
	var apiErr *geoserver.Error
	if errors.As(err, &apiErr) {
		fmt.Printf("typed error inspection:\n")
		fmt.Printf("  StatusCode = %d\n", apiErr.StatusCode)
		fmt.Printf("  Op         = %q\n", apiErr.Op)
		fmt.Printf("  URL        = %q\n", apiErr.URL)
		// Body is the truncated GeoServer response (max 8 KiB). Useful
		// for diagnostics; don't parse it for control flow — use the
		// sentinel match above instead.
		bodyPreview := string(apiErr.Body)
		if len(bodyPreview) > 120 {
			bodyPreview = bodyPreview[:120] + "..."
		}
		fmt.Printf("  Body       = %q\n", bodyPreview)
	}

	// 3. The pattern composes — chained sentinels work in switch:
	switch {
	case errors.Is(err, geoserver.ErrNotFound),
		errors.Is(err, geoserver.ErrConflict):
		fmt.Println("would treat 404 and 409 the same in this code path")
	}

	// 4. Successful calls produce no error; sanity check.
	if _, err := gs.GetWorkspacesContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "note: list-workspaces also failed — server unreachable? %v\n", err)
		return
	}
	fmt.Println("listing workspaces succeeded — server is reachable")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
