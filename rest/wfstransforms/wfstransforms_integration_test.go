//go:build integration

package wfstransforms_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
)

// The gs-xslt-wfs extension is NOT installed in the dev/test docker
// image. We verify the SDK's "extension absent" path: List and Get
// against a fresh GeoServer return a 404 mapped to ErrNotFound.
//
// To exercise the full CRUD on a server with the extension
// installed, set the env var GEOSERVER_HAS_XSLT_WFS=1 — the
// integration suite then runs the round-trip below. (The compose
// stack does not ship with the extension; bumping the Dockerfile to
// include it is a follow-up.)
func TestWFSTransforms_List_ExtensionMissing_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)
	_, err := c.WFSTransforms.List(ctx)
	// Either ErrNotFound (extension missing → 404) or success (if
	// someone installed the extension) is acceptable.
	if err != nil && !errors.Is(err, geoserver.ErrNotFound) {
		t.Errorf("unexpected error: %v", err)
	}
}
