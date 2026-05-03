package workspaces_test

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// ExampleClient_Get fetches a single workspace by name. Match
// [geoserver.ErrNotFound] via [errors.Is] to detect "doesn't exist"
// rather than checking for nil first — there is no Exists method on
// the v2 surface (see encoding/json and database/sql for the same
// idiom).
func ExampleClient_Get() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ws, err := c.Workspaces.Get(context.Background(), "topp")
	switch {
	case errors.Is(err, geoserver.ErrNotFound):
		fmt.Println("not found")
	case err != nil:
		// real failure
	default:
		fmt.Printf("%s isolated=%v\n", ws.Name, ws.Isolated)
	}
}

// ExampleClient_Iter ranges over every workspace using Go 1.23+
// range-over-func. The yielded error is non-nil only on the first
// iteration if the underlying List call fails.
func ExampleClient_Iter() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	for ws, err := range c.Workspaces.Iter(context.Background(), workspaces.ListOptions{}) {
		if err != nil {
			// fetch failed — bail out
			return
		}
		fmt.Println(ws.Name)
	}
}

// ExampleClient_Create registers a fresh workspace. A duplicate name
// surfaces as an [*geoserver.APIError] wrapping
// [geoserver.ErrConflict].
func ExampleClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	err := c.Workspaces.Create(context.Background(), &workspaces.Workspace{
		Name:     "my-workspace",
		Isolated: false,
	})
	if errors.Is(err, geoserver.ErrConflict) {
		fmt.Println("already exists")
	}
}

// ExampleClient_Update flips the workspace's Isolated flag. The
// pointer field on [workspaces.WorkspacePatch] lets callers
// distinguish "leave field alone" (nil) from "set to false" (&false).
func ExampleClient_Update() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	isolated := true
	_ = c.Workspaces.Update(context.Background(), "my-workspace",
		&workspaces.WorkspacePatch{Isolated: &isolated})
}

// ExampleClient_Delete removes a workspace and (with Recurse=true)
// every datastore, layer, and style it contains.
func ExampleClient_Delete() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Workspaces.Delete(context.Background(), "my-workspace",
		workspaces.DeleteOptions{Recurse: true})
}
