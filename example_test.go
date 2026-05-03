package geoserver_test

// Godoc-renderable examples. Each function attaches to a public symbol via
// the Go convention `ExampleSymbol` / `ExampleType_Method`, and renders on
// pkg.go.dev under that symbol's documentation page.
//
// These are also unit tests (run by `go test`), so they can't drift from
// the real API: a symbol rename or signature change breaks them at the
// next test run.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/hishamkaram/geoserver"
)

// ExampleNew shows the v1.1 functional-options constructor with a per-call
// timeout, basic auth (which is also accepted positionally for v1.0
// compatibility), and a custom user-agent.
func ExampleNew() {
	gs := geoserver.New("http://localhost:8080/geoserver/", "admin", "geoserver",
		geoserver.WithTimeout(10*time.Second),
		geoserver.WithUserAgent("my-tool/1.2.3"),
	)
	_ = gs // use gs.GetWorkspacesContext(ctx), etc.
	fmt.Println("client constructed")
	// Output: client constructed
}

// ExampleGeoServer_GetWorkspacesContext shows context-aware listing of
// workspaces against an httptest stub. Real callers point New at a live
// GeoServer URL.
func ExampleGeoServer_GetWorkspacesContext() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"workspaces":{"workspace":[{"name":"topp"},{"name":"sf"}]}}`)
	}))
	defer srv.Close()

	gs := geoserver.New(srv.URL+"/", "admin", "geoserver")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	workspaces, err := gs.GetWorkspacesContext(ctx)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, ws := range workspaces {
		fmt.Println(ws.Name)
	}
	// Output:
	// topp
	// sf
}

// ExampleError_Is shows the recommended pattern for matching GeoServer
// errors against package sentinel values via errors.Is.
func ExampleError_Is() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "workspace not found")
	}))
	defer srv.Close()

	gs := geoserver.New(srv.URL+"/", "admin", "geoserver")

	_, err := gs.GetWorkspaceContext(context.Background(), "missing")
	switch {
	case errors.Is(err, geoserver.ErrNotFound):
		fmt.Println("not found")
	case errors.Is(err, geoserver.ErrUnauthorized):
		fmt.Println("unauthorized")
	default:
		fmt.Println("other:", err)
	}
	// Output: not found
}

// ExampleACLRule_ToStrings demonstrates the round-trip helpers between an
// ACLRule struct and the wire-format pair (rule string, roles string)
// GeoServer's REST API expects.
func ExampleACLRule_ToStrings() {
	rule := geoserver.ACLRule{
		Workspace: "topp",
		Layer:     "states",
		Operation: geoserver.ACLOpRead,
		Roles:     []string{"viewer", "editor"},
	}
	ruleStr, rolesStr := rule.ToStrings()
	fmt.Println(ruleStr)
	fmt.Println(rolesStr)

	parsed, _ := geoserver.StringToACLRule(ruleStr, rolesStr)
	fmt.Println(parsed.Workspace, parsed.Layer, parsed.Operation)

	// Output:
	// topp.states.r
	// viewer,editor
	// topp states r
}
