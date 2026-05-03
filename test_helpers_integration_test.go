//go:build integration
// +build integration

package geoserver

import (
	"testing"

	"github.com/hishamkaram/geoserver/internal/testutil"
)

// mustPkgDir is a test helper that returns the package working directory or
// fails the test on error. Used to locate fixtures under testdata/.
//
// The gs argument is retained to preserve the existing call signature; the
// helper itself does not need it (the directory is the test's CWD, not
// state on the client).
func mustPkgDir(t *testing.T, gs *GeoServer) string {
	t.Helper()
	_ = gs
	dir, err := testutil.PkgDir()
	if err != nil {
		t.Fatalf("testutil.PkgDir: %v", err)
	}
	return dir
}
