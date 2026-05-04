// error-handling is a runnable v2 example: demonstrate how to match
// GeoServer errors with `errors.Is` against package sentinels and
// inspect the typed *geoserver.APIError via `errors.As`.
//
//	go run ./v2/examples/error-handling
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
		geoserver.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))),
	)
	if err != nil {
		fatal("construct client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. ErrNotFound on a missing workspace.
	_, err = c.Workspaces.Get(ctx, "definitely_not_a_real_workspace_v2")
	switch {
	case errors.Is(err, geoserver.ErrNotFound):
		fmt.Println("✓ ErrNotFound matched via errors.Is")
	case err == nil:
		fmt.Println("unexpected: workspace exists?")
	default:
		fmt.Printf("unexpected error: %v\n", err)
	}

	// 2. Inspect the underlying *APIError for the same call to read
	// status code, op, and the (capped) response body.
	_, err = c.Workspaces.Get(ctx, "still_not_real")
	var apiErr *geoserver.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("✓ *APIError: Op=%q Method=%q Status=%d BodyLen=%d\n",
			apiErr.Op, apiErr.Method, apiErr.StatusCode, len(apiErr.Body))
	}

	// 3. ErrConflict on a duplicate create.
	const dupName = "v2_examples_error_handling_dup"
	// best-effort cleanup
	_ = c.Workspaces.Delete(ctx, dupName, workspaces.DeleteOptions{Recurse: true})

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: dupName}); err != nil {
		fatal("first create: %v", err)
	}
	defer func() {
		_ = c.Workspaces.Delete(ctx, dupName, workspaces.DeleteOptions{Recurse: true})
	}()

	err = c.Workspaces.Create(ctx, &workspaces.Workspace{Name: dupName})
	switch {
	case errors.Is(err, geoserver.ErrConflict):
		fmt.Println("✓ ErrConflict matched on duplicate create")
	case err == nil:
		fmt.Println("unexpected: duplicate Create succeeded")
	default:
		fmt.Printf("unexpected error on duplicate create: %v\n", err)
	}

	// 4. ErrUnauthorized — point a fresh client at a deliberately wrong
	// password and probe a protected endpoint. Note no logger to keep
	// output clean.
	bad, err := geoserver.New(url,
		geoserver.WithBasicAuth(user, "absolutely-wrong-password"),
		geoserver.WithTimeout(5*time.Second),
	)
	if err != nil {
		fatal("construct bad-auth client: %v", err)
	}
	_, err = bad.Workspaces.List(ctx, workspaces.ListOptions{})
	switch {
	case errors.Is(err, geoserver.ErrUnauthorized):
		fmt.Println("✓ ErrUnauthorized matched on bad credentials")
	case err == nil:
		fmt.Println("unexpected: list succeeded with bad creds")
	default:
		fmt.Printf("unexpected auth error: %v\n", err)
	}

	// 5. The full sentinel set, for reference.
	fmt.Println()
	fmt.Println("Full sentinel set you can match with errors.Is:")
	for _, s := range []error{
		geoserver.ErrBadRequest,
		geoserver.ErrUnauthorized,
		geoserver.ErrForbidden,
		geoserver.ErrNotFound,
		geoserver.ErrMethodNotAllowed,
		geoserver.ErrConflict,
		geoserver.ErrUnsupportedMediaType,
		geoserver.ErrRateLimited,
		geoserver.ErrServerError,
		geoserver.ErrBadGateway,
		geoserver.ErrServiceUnavailable,
		geoserver.ErrGatewayTimeout,
	} {
		fmt.Printf("  - %v\n", s)
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
