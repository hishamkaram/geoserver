package imports_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/imports"
)

// ExampleClient_Create starts an import session that scans a
// directory of files and populates one task per discovered file.
// The Importer auto-creates target stores when none is named.
func ExampleClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	imp, err := c.Imports.Create(context.Background(), imports.ImportRequest{
		TargetWorkspace: "topp",
		Data: &imports.Data{
			Type:     imports.DataTypeDirectory,
			Location: "/srv/data/incoming/2026-05-03",
		},
	}, imports.CreateOptions{Execute: true})
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("Importer extension not installed")
		return
	}
	if err != nil {
		return
	}
	fmt.Printf("created session %d in state %s\n", imp.ID, imp.State)
}

// ExampleClient_Get_polling polls a session until it reaches a
// terminal state. Useful for the Async / Execute flow where the
// caller needs to know when the import finishes.
func ExampleClient_Get_polling() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		got, err := c.Imports.Get(context.Background(), 42)
		if err != nil {
			return
		}
		switch got.State {
		case imports.StateComplete, imports.StateCompleteError, imports.StateInitError:
			fmt.Printf("done: state=%s\n", got.State)
			return
		}
		time.Sleep(2 * time.Second)
	}
}

// ExampleClient_AddTask appends a task to an existing session — the
// typical pattern when the data set arrives in pieces.
func ExampleClient_AddTask() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_, _ = c.Imports.AddTask(context.Background(), 42, imports.TaskRequest{
		Source: &imports.Data{
			Type: imports.DataTypeFile,
			File: "/srv/data/incoming/late_arrival.shp",
		},
	})
}

// ExampleClient_Execute kicks off the import after configuring all
// tasks. Combine with [Client.Get] in a polling loop to await
// completion.
func ExampleClient_Execute() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	if err := c.Imports.Execute(context.Background(), 42); err != nil {
		fmt.Printf("Execute: %v\n", err)
	}
}
