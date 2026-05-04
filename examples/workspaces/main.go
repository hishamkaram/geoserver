// workspaces is a runnable v2 example: list, create, get, and delete a
// GeoServer workspace using the v2 functional-options constructor and
// flat *Client.Workspaces sub-client.
//
// Run with a compose stack at http://localhost:8080/geoserver/ (defaults
// from `make compose-up`):
//
//	go run ./v2/examples/workspaces
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	c, err := geoserver.New(url,
		geoserver.WithBasicAuth(user, pass),
		geoserver.WithTimeout(10*time.Second),
		geoserver.WithUserAgent("geoserver-v2-examples-workspaces/1.0"),
		geoserver.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))),
	)
	if err != nil {
		fatal("construct client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List existing workspaces.
	existing, err := c.Workspaces.List(ctx, workspaces.ListOptions{})
	if err != nil {
		fatal("list workspaces: %v", err)
	}
	fmt.Printf("found %d existing workspaces\n", len(existing))

	// Create a fresh workspace; clean up at the end.
	const name = "v2_examples_workspaces_demo"

	// Best-effort cleanup in case a previous run left it behind.
	_ = c.Workspaces.Delete(ctx, name, workspaces.DeleteOptions{Recurse: true})

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name}); err != nil {
		fatal("create workspace %q: %v", name, err)
	}
	fmt.Printf("created workspace %q\n", name)

	// Fetch the workspace we just created — sanity check.
	ws, err := c.Workspaces.Get(ctx, name)
	if err != nil {
		fatal("fetch workspace %q: %v", name, err)
	}
	fmt.Printf("fetched workspace: name=%q isolated=%t\n", ws.Name, ws.Isolated)

	// 404 on a non-existent workspace returns ErrNotFound via errors.Is.
	_, err = c.Workspaces.Get(ctx, "definitely_not_a_real_workspace_v2")
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("expected ErrNotFound for a missing workspace — verified")
	}

	// Iterate (range-over-func) — useful when you want to streaming-process
	// the list without materializing the full slice.
	count := 0
	for ws, iterErr := range c.Workspaces.Iter(ctx, workspaces.ListOptions{}) {
		if iterErr != nil {
			fatal("iter workspaces: %v", iterErr)
		}
		count++
		_ = ws
	}
	fmt.Printf("iterator yielded %d workspaces\n", count)

	// Clean up.
	if err := c.Workspaces.Delete(ctx, name, workspaces.DeleteOptions{Recurse: true}); err != nil {
		fatal("delete workspace %q: %v", name, err)
	}
	fmt.Printf("deleted workspace %q\n", name)
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
