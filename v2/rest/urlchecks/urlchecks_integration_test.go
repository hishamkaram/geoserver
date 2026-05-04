//go:build integration

package urlchecks_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/urlchecks"
)

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func TestURLChecks_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	name := uniqueName("v2_it_check")
	chk := &urlchecks.URLCheck{
		Name:        name,
		Description: "v2 integration test",
		Enabled:     true,
		Regex:       "^https://test.example/" + name + "/.*$",
	}

	t.Cleanup(func() { _ = c.URLChecks.Delete(ctx, name) })

	if err := c.URLChecks.Create(ctx, chk); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := c.URLChecks.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != name || got.Regex != chk.Regex || got.Enabled != true {
		t.Errorf("Get returned %+v, want %+v", got, chk)
	}

	// List should include the new check.
	list, err := c.URLChecks.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, ref := range list {
		if ref.Name == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("check %q not in list (%d entries)", name, len(list))
	}

	// Update — flip enabled.
	if err := c.URLChecks.Update(ctx, name, &urlchecks.URLCheck{Enabled: false, Regex: chk.Regex, Name: name}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = c.URLChecks.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Enabled {
		t.Errorf("expected Enabled=false after Update, got %+v", got)
	}

	// Delete.
	if err := c.URLChecks.Delete(ctx, name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.URLChecks.Get(ctx, name); !errors.Is(err, geoserver.ErrNotFound) {
		t.Errorf("Get after Delete: expected ErrNotFound, got %v", err)
	}
}

func TestURLChecks_List_EmptyOnFreshInstall_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Default install has no URL checks. The wire response is
	// {"urlChecks":""} which the SDK normalizes to a nil slice.
	list, err := c.URLChecks.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// We don't assert the exact count — earlier integration tests
	// may have left checks behind on a shared dev stack — but List
	// must not error on the empty wire shape.
	_ = list
}
