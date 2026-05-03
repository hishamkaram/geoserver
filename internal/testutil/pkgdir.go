// Package testutil contains helpers used only by the integration test suite.
// External code must not import this package — the import path lives under
// internal/ to enforce the rule at compile time.
package testutil

import "path/filepath"

// PkgDir returns the absolute path of the current working directory, which
// for `go test`-run tests resolves to the test's package directory. Used by
// integration tests to locate fixtures under testdata/.
func PkgDir() (string, error) {
	return filepath.Abs("./")
}
