package geoserver_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// ExampleNew constructs a Client against a local GeoServer instance.
// All other configuration (auth, timeout, logging) is layered via
// [geoserver.Option] values.
func ExampleNew() {
	c, err := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"),
		geoserver.WithTimeout(10*time.Second),
	)
	if err != nil {
		// In real code: handle the error. New only fails on
		// invalid serverURL or option misconfiguration.
		panic(err)
	}
	_ = c
}

// ExampleNew_basicAuth shows the typical credential setup for an
// on-premise GeoServer instance.
func ExampleNew_basicAuth() {
	_, _ = geoserver.New(
		"https://geoserver.example.com/geoserver",
		geoserver.WithBasicAuth("admin", os.Getenv("GEOSERVER_PASSWORD")),
	)
}

// ExampleNew_bearerToken shows how to authenticate against a
// GeoServer fronted by an OAuth2 / JWT proxy.
func ExampleNew_bearerToken() {
	_, _ = geoserver.New(
		"https://geoserver.example.com/geoserver",
		geoserver.WithBearerToken(os.Getenv("GEOSERVER_TOKEN")),
	)
}

// ExampleNew_logging wires the client's slog handler to text output
// on stderr at debug level — every HTTP request is logged.
func ExampleNew_logging() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	_, _ = geoserver.New(
		"http://localhost:8080/geoserver",
		geoserver.WithLogger(logger),
	)
}

// Example_errorHandling shows the two idiomatic ways to inspect a
// returned error: [errors.Is] against the package sentinels for
// status-code matching, and [errors.As] to a [*geoserver.APIError]
// for the full request context.
func Example_errorHandling() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_, err := c.Workspaces.Get(context.Background(), "no-such-workspace")
	switch {
	case err == nil:
		// found
	case errors.Is(err, geoserver.ErrNotFound):
		fmt.Println("not found — create it instead")
	case errors.Is(err, geoserver.ErrUnauthorized):
		fmt.Println("credentials rejected")
	default:
		// Inspect the typed error for diagnostics.
		var apiErr *geoserver.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("%s %s -> %d\n", apiErr.Method, apiErr.URL, apiErr.StatusCode)
		}
	}
}
