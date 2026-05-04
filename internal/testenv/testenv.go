// Package testenv holds shared helpers for v2 integration tests. The
// helpers run against a real GeoServer + PostGIS stack (boot via
// `make compose-up` from the repo root) and are gated behind the
// integration build tag at each call site.
//
// External code must not import this package — it lives under
// internal/ to keep the helpers test-only.
package testenv

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// Defaults match `make compose-up`.
const (
	DefaultURL  = "http://localhost:8080/geoserver/"
	DefaultUser = "admin"
	DefaultPass = "geoserver"

	// PostGIS connection inside the compose network. Tests connecting
	// the SDK to the same PostGIS instance use these values; tests
	// running outside the compose network would need to substitute
	// the host-mapped port (5436) — but our integration runs only
	// from inside the workflow, where the GeoServer container talks
	// to the PostGIS container by service name.
	DBHost = "postgis"
	DBPort = 5432
	DBName = "gis"
	DBUser = "golang"
	DBPass = "golang"
)

// counter ensures uniqueness across parallel test runs.
var counter uint64

// NewClient constructs a v2 client pointed at the env-configured
// GeoServer with sane integration-test defaults.
func NewClient(t *testing.T) *geoserver.Client {
	t.Helper()
	url := envOr("GEOSERVER_URL", DefaultURL)
	user := envOr("GEOSERVER_USER", DefaultUser)
	pass := envOr("GEOSERVER_PASS", DefaultPass)

	c, err := geoserver.New(url,
		geoserver.WithBasicAuth(user, pass),
		geoserver.WithTimeout(60*time.Second),
		geoserver.WithUserAgent("geoserver-v2-integration/1.0"),
	)
	if err != nil {
		t.Fatalf("testenv.NewClient: %v", err)
	}
	return c
}

// Context returns a 2-minute-bounded context cancelled in t.Cleanup.
func Context(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// UniqueName returns a test-unique resource name with the given
// prefix. Names include the test name (lower-cased, sanitized) and
// an atomic counter so parallel tests don't collide.
//
// GeoServer rejects names containing '/' or whitespace; the
// sanitizer strips them.
func UniqueName(t *testing.T, prefix string) string {
	t.Helper()
	n := atomic.AddUint64(&counter, 1)
	tname := sanitize(t.Name())
	return fmt.Sprintf("v2_it_%s_%s_%d", prefix, tname, n)
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+'a'-'A')
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
