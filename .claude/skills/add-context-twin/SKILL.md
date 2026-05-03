---
description: Adds a Context-aware sibling method to a GeoServer client method, following the established v1.1 pattern. Use when adding a new exported method on *GeoServer or porting an old method that lacks a *Context variant. Outputs the full code shape (non-context wrapper + Context impl + interface entry + unit test stub) for the caller to apply.
argument-hint: [method-name] [resource-file]
disable-model-invocation: true
---

# Add `*Context` twin to `$0`

Target file: **`$1`**.
Target method name (non-Context form): **`$0`** — the Context sibling will be **`$0Context`**.

This skill is manually invoked because it expects arguments. Run as:

```
/add-context-twin GetFoo workspaces.go
```

## What this skill does

Generates the four pieces every new `*GeoServer` method needs to comply with the v1.1.x *Context twin pattern. Read the canonical reference at `workspaces.go:16-38,57-79` (the `CreateWorkspace` / `CreateWorkspaceContext` pair) before applying.

## 1. The non-context wrapper

Add this in `$1`. The body is always a single line that delegates with `context.Background()`:

```go
// $0 is the context.Background()-using shim around [GeoServer.$0Context].
//
// Deprecated: prefer [GeoServer.$0Context] in new code.
func (g *GeoServer) $0(/* args */) (/* returns */) {
    return g.$0Context(context.Background(), /* args */)
}
```

Keep the same exported argument and return signatures as the legacy v1.0 method (or whatever the new method's contract is, if it's brand-new). The wrapper exists for source-compatibility with v1.0 callers who don't pass a `ctx`.

## 2. The Context-aware sibling

Right below the wrapper, the real implementation:

```go
// $0Context is the context-aware variant of [GeoServer.$0].
func (g *GeoServer) $0Context(ctx context.Context, /* args */) (/* returns */) {
    targetURL := g.<urlBuilder>(/* segments */)
    httpRequest := HTTPRequest{
        Method: getMethod, // or postMethod / putMethod / deleteMethod
        Accept: jsonType,
        URL:    targetURL,
        Query:  nil,
    }
    response, responseCode := g.DoRequestContext(ctx, httpRequest)
    if responseCode != statusOk {
        g.logger.Error(string(response))
        return /* zero values */, g.GetError(responseCode, response)
    }
    /* deserialize response, return */
    return
}
```

**Constraints:**
- The first parameter is `ctx context.Context`.
- Use `g.DoRequestContext(ctx, ...)` — never `g.DoRequest` (which exists only as the `context.Background()` shim).
- URL building: prefer the per-resource helper if one exists (`g.workspacesURL`, `g.stylesURL`, etc.) over raw `g.ParseURL("rest", ...)`. Both are correct; helpers reduce duplication.
- Errors via `g.GetError(responseCode, response)` — that returns a `*Error` with the right sentinel mapped via `errors.Is`.

## 3. Add the new method to the parallel `*ServiceWithContext` interface

Find the resource's interface declaration in `$1` (e.g., `WorkspaceServiceWithContext` near the top). Add an entry mirroring `$0Context`. Also add the non-context form to the legacy `*Service` interface so the `Catalog` interface (in `catalog.go`) keeps embedding both cleanly.

## 4. Unit test (`$1` → strip `.go` → append `_unit_test.go`)

Pattern from `workspaces_unit_test.go`:

```go
func Test$0Context_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // assert request shape: method, URL path, headers
        if r.Method != http.MethodGet { t.Fatalf("method = %s", r.Method) }
        // ...
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{ /* canned response */ }`))
    }))
    defer server.Close()

    gs := New(server.URL+"/", "u", "p")
    got, err := gs.$0Context(context.Background(), /* args */)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    // assert got is what we expect
}

func Test$0Context_NotFound(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        _, _ = w.Write([]byte(`{"abstract":"x","details":"y"}`))
    }))
    defer server.Close()

    gs := New(server.URL+"/", "u", "p")
    _, err := gs.$0Context(context.Background(), /* args */)
    if !errors.Is(err, ErrNotFound) {
        t.Fatalf("got %v, want ErrNotFound", err)
    }
}
```

At minimum cover: 2xx happy path, 404, plus one of {401, 403, 409, 500} appropriate to the resource.

## 5. Sanity checks

After applying:

- `go build ./...` clean.
- `go vet ./...` clean.
- `make test-unit` green.
- The reference pair (`workspaces.go:16-38,57-79`) and your new pair both compile cleanly under `golangci-lint run`.
