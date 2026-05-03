// workspaces is a runnable example: list, create, and delete a GeoServer
// workspace using the v1.1 functional-options constructor and *Context
// method variants.
//
// Run with a compose stack at http://localhost:8080/geoserver/ (defaults
// from `make compose-up`):
//
//	go run ./examples/workspaces
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hishamkaram/geoserver"
)

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	gs := geoserver.New(url, user, pass,
		geoserver.WithTimeout(10*time.Second),
		geoserver.WithUserAgent("geoserver-examples-workspaces/1.0"),
		geoserver.WithLogger(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List existing workspaces.
	workspaces, err := gs.GetWorkspacesContext(ctx)
	if err != nil {
		fatal("list workspaces: %v", err)
	}
	fmt.Printf("found %d existing workspaces\n", len(workspaces))

	// Create a fresh workspace; clean up at the end.
	const name = "examples_workspaces_demo"

	// Best-effort delete in case a previous run left it behind.
	_, _ = gs.DeleteWorkspaceContext(ctx, name, true)

	created, err := gs.CreateWorkspaceContext(ctx, name)
	if err != nil {
		fatal("create workspace %q: %v", name, err)
	}
	fmt.Printf("created workspace %q (created=%t)\n", name, created)

	// Fetch the workspace we just created — sanity check.
	ws, err := gs.GetWorkspaceContext(ctx, name)
	if err != nil {
		fatal("fetch workspace %q: %v", name, err)
	}
	fmt.Printf("fetched workspace: name=%q isolated=%t\n", ws.Name, ws.Isolated)

	// 404 on a non-existent workspace returns ErrNotFound via errors.Is.
	_, err = gs.GetWorkspaceContext(ctx, "definitely_not_a_real_workspace")
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("expected ErrNotFound for a missing workspace — verified")
	}

	// Clean up.
	deleted, err := gs.DeleteWorkspaceContext(ctx, name, true)
	if err != nil {
		fatal("delete workspace %q: %v", name, err)
	}
	fmt.Printf("deleted workspace %q (deleted=%t)\n", name, deleted)
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
