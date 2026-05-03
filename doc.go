// Package geoserver is a Go client for the GeoServer REST API.
//
// # Status
//
// This is the v1 line — stable and production-ready. v1.1 kept the v1.0
// public surface intact (every existing method shape still compiles and
// behaves the same) while adding context-aware siblings, typed errors,
// stdlib log/slog logging, and a functional-options [New] constructor.
// Existing v1.0 callers can upgrade to any v1.x release with only a
// go.mod bump.
//
// For the next-generation sub-client SDK, see
// [github.com/hishamkaram/geoserver/v2] (preview, currently
// v2.0.0-alpha.4).
//
// # Quick start
//
//	gs := geoserver.New(
//	    "http://localhost:8080/geoserver/",
//	    "admin", "geoserver",
//	    geoserver.WithTimeout(15*time.Second),
//	    geoserver.WithLogger(slog.NewTextHandler(os.Stderr, nil)),
//	)
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	if _, err := gs.CreateWorkspaceContext(ctx, "demo"); err != nil &&
//	    !errors.Is(err, geoserver.ErrConflict) {
//	    return err
//	}
//
// # Design
//
//   - *Context twin pattern. Every public method has a Context-aware
//     sibling that takes [context.Context] as its first argument; the
//     non-context name is a one-line wrapper that calls the *Context
//     version with [context.Background]. Prefer the *Context form in
//     new code so cancellation and deadlines propagate.
//   - Typed errors. Every HTTP error is a [*Error] wrapping one of the
//     package sentinels ([ErrNotFound], [ErrConflict], …). Match with
//     [errors.Is] and [errors.As]; do not compare error strings.
//   - Functional options. Configure the HTTP client, timeout, logger,
//     and user-agent at construction via [Option] helpers
//     ([WithHTTPClient], [WithTimeout], [WithLogger], [WithUserAgent],
//     [WithBasicAuth]).
//   - log/slog logging. The library logs through stdlib [log/slog],
//     silent by default. Configure via [WithLogger]. No third-party
//     logger dependency.
//
// # Concurrency
//
// A *GeoServer constructed via [New] is safe for concurrent reads
// (i.e. concurrent calls to its methods); however, mutating its
// exported fields after construction is NOT safe. Construct once and
// treat as immutable. The v2 line replaces the exported-fields model
// with a fully private, immutable client.
//
// # Versions
//
// Supported GeoServer versions: 2.27 (LTS) and 2.28 (current stable).
// GeoServer 3.0 support is tracked for v2 once Tomcat 11 / Jakarta EE
// / ImageN settle. See the project ROADMAP for details.
package geoserver
