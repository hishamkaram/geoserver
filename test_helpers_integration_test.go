//go:build integration
// +build integration

package geoserver

import "testing"

// mustPkgDir is a test helper that returns the package working directory or
// fails the test on error. Used to locate fixtures under testdata/.
func mustPkgDir(t *testing.T, gs *GeoServer) string {
	t.Helper()
	dir, err := gs.getGoGeoserverPackageDir()
	if err != nil {
		t.Fatalf("getGoGeoserverPackageDir: %v", err)
	}
	return dir
}
