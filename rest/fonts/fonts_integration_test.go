//go:build integration

package fonts_test

import (
	"strings"
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
)

func TestFonts_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.Fonts.List(ctx)
	if err != nil {
		t.Fatalf("Fonts.List: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one font, got empty list")
	}

	// Sanity-check that the list is non-empty and entries are strings
	// (not all-empty). The exact set is JVM- and OS-dependent, so we
	// only assert "at least one non-empty entry."
	var nonEmpty int
	for _, f := range got {
		if strings.TrimSpace(f) != "" {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		t.Fatalf("every font name was empty: %v", got)
	}
}
