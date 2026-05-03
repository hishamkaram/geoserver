//go:build integration

package namespaces_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/namespaces"
)

func TestNamespaces_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	prefix := testenv.UniqueName(t, "ns")
	uri := "http://example.com/" + prefix

	t.Cleanup(func() {
		_ = c.Namespaces.Delete(ctx, prefix)
	})

	if err := c.Namespaces.Create(ctx, &namespaces.Namespace{
		Prefix: prefix, URI: uri,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := c.Namespaces.Get(ctx, prefix)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Prefix != prefix || got.URI != uri {
		t.Fatalf("Namespace = %+v", got)
	}

	// List includes it.
	all, err := c.Namespaces.List(ctx, namespaces.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, n := range all {
		if n.Prefix == prefix {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("namespace %q not found in List", prefix)
	}

	// Update — change URI.
	newURI := "http://example.com/new/" + prefix
	if err := c.Namespaces.Update(ctx, prefix, &namespaces.Patch{URI: &newURI}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = c.Namespaces.Get(ctx, prefix)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.URI != newURI {
		t.Errorf("URI = %q, want %q", got.URI, newURI)
	}

	if err := c.Namespaces.Delete(ctx, prefix); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Namespaces.Get(ctx, prefix); !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}
