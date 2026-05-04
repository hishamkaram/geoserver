// Package imports is the v2 sub-client for the GeoServer Importer
// extension at `/rest/imports`. The Importer collapses the
// workspace → store → featuretype/layer dance into a single import
// session — the standard tool for batch ingest, migrations, and
// nightly drop-and-republish workflows.
//
// The extension is **not installed by default**; this client's
// integration tests skip cleanly when the endpoint returns 404.
//
// Scope this package covers:
//
//   - Session lifecycle: Create, List, Iter, Get, Delete, Execute.
//   - Task management: ListTasks, AddTask, GetTask, UpdateTask,
//     DeleteTask.
//
// Scope deferred to follow-up PRs:
//
//   - Per-task Layer sub-resource (GET / PUT
//     `/rest/imports/{id}/tasks/{taskId}/layer`).
//   - Per-task Transforms sub-resource (CRS reprojection, attribute
//     renaming).
//   - Per-task Data sub-resource (file listing, file deletion).
//   - Database and mosaic data types (file / directory / remote
//     are exposed; database and mosaic require additional
//     connection-parameter shapes — add when a real caller asks).
package imports

// State is the documented import-session lifecycle state.
type State string

// Documented import states (from the Importer extension docs).
const (
	StateInit          State = "INIT"
	StateInitError     State = "INIT_ERROR"
	StatePending       State = "PENDING"
	StateReady         State = "READY"
	StateRunning       State = "RUNNING"
	StateComplete      State = "COMPLETE"
	StateCompleteError State = "COMPLETE_ERROR"
)

// DataType selects the Importer data-source kind. Only `file`,
// `directory`, and `remote` are exposed in this package version;
// `database` and `mosaic` need additional fields and are deferred.
type DataType string

// Data source kinds.
const (
	DataTypeFile      DataType = "file"
	DataTypeDirectory DataType = "directory"
	DataTypeRemote    DataType = "remote"
)

// Import is the session document.
//
// `Tasks` is populated only on certain endpoints (Get with detail,
// or freshly after Create with auto-population). On the bare List
// endpoint the entries typically only carry `ID` and `State`.
type Import struct {
	ID    int64  `json:"id,omitempty"`
	State State  `json:"state,omitempty"`
	Href  string `json:"href,omitempty"`

	TargetWorkspace *WorkspaceRef `json:"targetWorkspace,omitempty"`
	TargetStore     *StoreRef     `json:"targetStore,omitempty"`
	Data            *Data         `json:"data,omitempty"`

	Tasks []Task `json:"tasks,omitempty"`
}

// WorkspaceRef wraps the workspace name in the Importer's nested
// shape: `{"workspace":{"name":"<name>"}}`.
type WorkspaceRef struct {
	Workspace WorkspaceName `json:"workspace"`
}

// WorkspaceName carries the workspace's name.
type WorkspaceName struct {
	Name string `json:"name"`
}

// StoreRef wraps the datastore name in the Importer's nested shape:
// `{"dataStore":{"name":"<name>"}}`.
type StoreRef struct {
	DataStore StoreName `json:"dataStore"`
}

// StoreName carries the datastore's name.
type StoreName struct {
	Name string `json:"name"`
}

// Data describes the import's data source. The field set varies by
// `Type`:
//
//   - DataTypeFile: set `File` to the absolute local path or HTTP URL
//     of a single file.
//   - DataTypeDirectory: set `Location` to the directory path.
//   - DataTypeRemote: set `Location` to the remote URL.
type Data struct {
	Type     DataType `json:"type"`
	File     string   `json:"file,omitempty"`
	Location string   `json:"location,omitempty"`
}

// Task is one per-file unit of work inside an import session.
type Task struct {
	ID     int64  `json:"id,omitempty"`
	Href   string `json:"href,omitempty"`
	State  State  `json:"state,omitempty"`
	Source *Data  `json:"source,omitempty"`

	UpdateMode string `json:"updateMode,omitempty"`
}

// ImportRequest is the create-session payload for
// [Client.Create].
type ImportRequest struct {
	// TargetWorkspace is the destination workspace name. Optional —
	// when empty, the importer infers a target from the data source.
	TargetWorkspace string

	// TargetStore is the destination datastore name. Optional —
	// when empty, the importer infers or auto-creates one.
	TargetStore string

	// Data describes the data source. Optional — the importer can
	// also discover data from the request alone in some flows.
	Data *Data
}

// CreateOptions controls a [Client.Create] call.
type CreateOptions struct {
	// Async requests asynchronous create-session behavior. The
	// server returns immediately with the session in INIT state;
	// the caller polls Get until the state advances. Default:
	// synchronous (the call blocks until the session is READY).
	Async bool

	// Execute starts the session immediately after auto-population
	// completes. Equivalent to a manual Execute call after Create.
	Execute bool
}

// TaskRequest is the body for [Client.AddTask] — append a task to
// an existing session.
type TaskRequest struct {
	// Source describes the file/directory/URL the task ingests.
	Source *Data
}

// importEnvelope is the wire wrapper for create/get/update bodies:
// `{"import":{...}}`.
type importEnvelope struct {
	Import *Import `json:"import"`
}

// importsListEnvelope is the wire wrapper for the list response:
// `{"imports":[{...}, ...]}`.
type importsListEnvelope struct {
	Imports []Import `json:"imports"`
}

// taskEnvelope is the wire wrapper for task POST/PUT/GET bodies:
// `{"task":{...}}`.
type taskEnvelope struct {
	Task *Task `json:"task"`
}

// tasksListEnvelope is the wire wrapper for the per-session task
// list: `{"tasks":[{...}, ...]}`.
type tasksListEnvelope struct {
	Tasks []Task `json:"tasks"`
}
