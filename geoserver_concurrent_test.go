package geoserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// TestClient_ConcurrentRequests asserts that a single *Client is safe
// to share across goroutines. Designed to fail under `go test -race`
// if a future change introduces shared mutable state on Client,
// clientCore, or any sub-client.
func TestClient_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/about/version":
			_, _ = w.Write([]byte(`{"about":{"resource":[]}}`))
		case "/rest/workspaces/topp":
			_, _ = w.Write([]byte(`{"workspace":{"name":"topp","isolated":false}}`))
		case "/rest/namespaces/topp":
			_, _ = w.Write([]byte(`{"namespace":{"prefix":"topp","uri":"http://example.com/topp","isolated":false}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := geoserver.New(srv.URL, geoserver.WithBasicAuth("admin", "geoserver"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const goroutines = 50
	const callsPer = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for range callsPer {
				if err := c.About.Ping(ctx); err != nil {
					t.Errorf("Ping: %v", err)
					return
				}
				if _, err := c.About.Version(ctx); err != nil {
					t.Errorf("Version: %v", err)
					return
				}
				if _, err := c.Workspaces.Get(ctx, "topp"); err != nil {
					t.Errorf("Workspaces.Get: %v", err)
					return
				}
				if _, err := c.Namespaces.Get(ctx, "topp"); err != nil {
					t.Errorf("Namespaces.Get: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
