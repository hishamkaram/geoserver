package imports_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/imports"
)

func newTestClient(t *testing.T, srv *httptest.Server) *geoserver.Client {
	t.Helper()
	c, err := geoserver.New(srv.URL,
		geoserver.WithBasicAuth("admin", "geoserver"),
		geoserver.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestCreate_BodyShape(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q", r.Method)
		}
		if r.URL.Path != "/rest/imports" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		captured, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"import":{"id":42,"state":"READY"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	imp, err := c.Imports.Create(context.Background(), imports.ImportRequest{
		TargetWorkspace: "topp",
		TargetStore:     "states_pg",
		Data: &imports.Data{
			Type: imports.DataTypeFile,
			File: "/data/states.shp",
		},
	}, imports.CreateOptions{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if imp.ID != 42 || imp.State != imports.StateReady {
		t.Errorf("Import = %+v", imp)
	}

	body := string(captured)
	if !strings.HasPrefix(body, `{"import":`) {
		t.Errorf("envelope wrong: %q", body)
	}
	if !strings.Contains(body, `"workspace":{"name":"topp"}`) {
		t.Errorf("body missing nested workspace shape: %q", body)
	}
	if !strings.Contains(body, `"dataStore":{"name":"states_pg"}`) {
		t.Errorf("body missing nested dataStore shape: %q", body)
	}
	if !strings.Contains(body, `"type":"file"`) {
		t.Errorf("body missing data.type: %q", body)
	}
	if !strings.Contains(body, `"file":"/data/states.shp"`) {
		t.Errorf("body missing data.file: %q", body)
	}
}

func TestCreate_OmitsOptionalFields(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"import":{"id":1,"state":"INIT"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, _ = c.Imports.Create(context.Background(), imports.ImportRequest{}, imports.CreateOptions{})

	body := string(captured)
	if strings.Contains(body, "targetWorkspace") {
		t.Errorf("targetWorkspace should be omitted when empty: %q", body)
	}
	if strings.Contains(body, "targetStore") {
		t.Errorf("targetStore should be omitted when empty: %q", body)
	}
	if strings.Contains(body, `"data":`) {
		t.Errorf("data should be omitted when nil: %q", body)
	}
}

func TestCreate_AsyncExecuteQuery(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.RawQuery
		_, _ = io.WriteString(w, `{"import":{"id":1,"state":"INIT"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, _ = c.Imports.Create(context.Background(), imports.ImportRequest{}, imports.CreateOptions{
		Async:   true,
		Execute: true,
	})
	if !strings.Contains(captured, "async=true") {
		t.Errorf("query missing async: %q", captured)
	}
	if !strings.Contains(captured, "exec=true") {
		t.Errorf("query missing exec: %q", captured)
	}
}

func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/imports" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"imports":[
			{"id":1,"state":"COMPLETE"},
			{"id":2,"state":"PENDING"}
		]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Imports.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].State != imports.StatePending {
		t.Errorf("List = %+v", got)
	}
}

func TestIter_RangeOverFunc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"imports":[{"id":1},{"id":2},{"id":3}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var ids []int64
	for im, err := range c.Imports.Iter(context.Background()) {
		if err != nil {
			t.Fatalf("Iter: %v", err)
		}
		ids = append(ids, im.ID)
	}
	if len(ids) != 3 {
		t.Errorf("Iter ids = %+v", ids)
	}
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/imports/42" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"import":{"id":42,"state":"COMPLETE",
			"targetWorkspace":{"workspace":{"name":"topp"}}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Imports.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != 42 || got.State != imports.StateComplete {
		t.Errorf("Get = %+v", got)
	}
	if got.TargetWorkspace == nil || got.TargetWorkspace.Workspace.Name != "topp" {
		t.Errorf("TargetWorkspace = %+v", got.TargetWorkspace)
	}
}

func TestExecute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q", r.Method)
		}
		if r.URL.Path != "/rest/imports/42" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Imports.Execute(context.Background(), 42); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %q", r.Method)
		}
		if r.URL.Path != "/rest/imports/42" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Imports.Delete(context.Background(), 42); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestListTasks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/imports/42/tasks" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"tasks":[
			{"id":0,"state":"READY"},
			{"id":1,"state":"PENDING"}
		]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Imports.ListTasks(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(got) != 2 || got[1].State != imports.StatePending {
		t.Errorf("ListTasks = %+v", got)
	}
}

func TestAddTask(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/imports/42/tasks" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		captured, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"task":{"id":7,"state":"READY"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Imports.AddTask(context.Background(), 42, imports.TaskRequest{
		Source: &imports.Data{Type: imports.DataTypeFile, File: "/data/extra.shp"},
	})
	if err != nil {
		t.Fatalf("AddTask: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("AddTask returned %+v", got)
	}
	body := string(captured)
	if !strings.HasPrefix(body, `{"task":`) {
		t.Errorf("envelope wrong: %q", body)
	}
	if !strings.Contains(body, `"file":"/data/extra.shp"`) {
		t.Errorf("body missing file: %q", body)
	}
}

func TestAddTask_NilSource(t *testing.T) {
	c, _ := geoserver.New("http://localhost:8080", geoserver.WithBasicAuth("u", "p"))
	if _, err := c.Imports.AddTask(context.Background(), 42, imports.TaskRequest{}); err == nil {
		t.Errorf("expected error for nil Source")
	}
}

func TestGetTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/imports/42/tasks/3" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"task":{"id":3,"state":"READY","updateMode":"CREATE"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Imports.GetTask(context.Background(), 42, 3)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.UpdateMode != "CREATE" {
		t.Errorf("UpdateMode = %q", got.UpdateMode)
	}
}

func TestUpdateTask(t *testing.T) {
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method = %q", r.Method)
		}
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Imports.UpdateTask(context.Background(), 42, 3, &imports.Task{
		UpdateMode: "APPEND",
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	body := string(captured)
	if !strings.Contains(body, `"updateMode":"APPEND"`) {
		t.Errorf("body missing updateMode: %q", body)
	}
}

func TestDeleteTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %q", r.Method)
		}
		if r.URL.Path != "/rest/imports/42/tasks/3" {
			t.Errorf("Path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Imports.DeleteTask(context.Background(), 42, 3); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
}

func TestNotFound_NoExtensionInstalled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Imports.List(context.Background())
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
