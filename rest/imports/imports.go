package imports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strconv"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
}

// Client is the v2 Importer sub-client.
//
//	imp, err := c.Imports.Create(ctx, imports.ImportRequest{
//	    TargetWorkspace: "topp",
//	    Data: &imports.Data{Type: imports.DataTypeFile,
//	                       File: "/data/states.zip"},
//	})
//	c.Imports.Execute(ctx, imp.ID)
//
// Construct via the parent [*geoserver.Client]; do not call [New]
// directly outside the root package's wiring.
type Client struct {
	core Core
}

// New constructs the sub-client.
func New(core Core) *Client { return &Client{core: core} }

// ----- Session lifecycle -----

// Create starts a new import session. Use [CreateOptions.Async] to
// return immediately while the importer auto-populates tasks in the
// background; use [CreateOptions.Execute] to also kick off the
// session immediately after population.
//
// `req.TargetWorkspace`, `req.TargetStore`, and `req.Data` are all
// optional — the importer fills in reasonable defaults when omitted,
// per the data source.
func (c *Client) Create(ctx context.Context, req ImportRequest, opts CreateOptions) (*Import, error) {
	const op = "Imports.Create"
	u, err := c.core.URL("rest", "imports")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	body := importEnvelope{Import: &Import{
		TargetWorkspace: targetWorkspaceRef(req),
		TargetStore:     targetStoreRef(req),
		Data:            req.Data,
	}}
	query := buildCreateQuery(opts)

	var resp importEnvelope
	if err := c.core.Do(ctx, op, http.MethodPost, u, body, query, &resp); err != nil {
		return nil, err
	}
	if resp.Import == nil {
		return nil, errors.New(op + ": empty response import")
	}
	return resp.Import, nil
}

// targetWorkspaceRef returns the nested-shape WorkspaceRef for a
// non-empty workspace name, nil otherwise.
func targetWorkspaceRef(req ImportRequest) *WorkspaceRef {
	if req.TargetWorkspace == "" {
		return nil
	}
	return &WorkspaceRef{Workspace: WorkspaceName{Name: req.TargetWorkspace}}
}

// targetStoreRef returns the nested-shape StoreRef for a non-empty
// store name, nil otherwise.
func targetStoreRef(req ImportRequest) *StoreRef {
	if req.TargetStore == "" {
		return nil
	}
	return &StoreRef{DataStore: StoreName{Name: req.TargetStore}}
}

// buildCreateQuery assembles the optional query params for Create.
func buildCreateQuery(opts CreateOptions) map[string]string {
	if !opts.Async && !opts.Execute {
		return nil
	}
	q := map[string]string{}
	if opts.Async {
		q["async"] = "true"
	}
	if opts.Execute {
		q["exec"] = "true"
	}
	return q
}

// List returns every import session GeoServer knows about (active
// and recently-completed; the server prunes long-finished sessions).
func (c *Client) List(ctx context.Context) ([]Import, error) {
	const op = "Imports.List"
	u, err := c.core.URL("rest", "imports")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp importsListEnvelope
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Imports, nil
}

// Iter is the range-over-func form of [Client.List].
func (c *Client) Iter(ctx context.Context) iter.Seq2[Import, error] {
	return func(yield func(Import, error) bool) {
		ims, err := c.List(ctx)
		if err != nil {
			yield(Import{}, err)
			return
		}
		for _, im := range ims {
			if !yield(im, nil) {
				return
			}
		}
	}
}

// Get fetches a single import session by id.
func (c *Client) Get(ctx context.Context, id int64) (*Import, error) {
	const op = "Imports.Get"
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(id, 10))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp importEnvelope
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Import == nil {
		return nil, errors.New(op + ": empty response import")
	}
	return resp.Import, nil
}

// Execute kicks off (or restarts) the named session. The importer
// runs the session asynchronously; poll [Client.Get] for state
// transitions (PENDING → RUNNING → COMPLETE).
func (c *Client) Execute(ctx context.Context, id int64) error {
	const op = "Imports.Execute"
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(id, 10))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}

// Delete removes an import session. Once deleted, the server forgets
// the session — pending tasks are dropped; completed-task results
// (the published catalog entries) remain.
func (c *Client) Delete(ctx context.Context, id int64) error {
	const op = "Imports.Delete"
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(id, 10))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// ----- Task management -----

// ListTasks returns every task in the named session.
func (c *Client) ListTasks(ctx context.Context, importID int64) ([]Task, error) {
	const op = "Imports.ListTasks"
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(importID, 10), "tasks")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp tasksListEnvelope
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// AddTask appends a task to an existing session — useful when the
// data source is a directory with many files and the caller wants
// to add one at a time.
func (c *Client) AddTask(ctx context.Context, importID int64, req TaskRequest) (*Task, error) {
	const op = "Imports.AddTask"
	if req.Source == nil {
		return nil, errors.New(op + ": nil Source")
	}
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(importID, 10), "tasks")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	body := taskEnvelope{Task: &Task{Source: req.Source}}
	var resp taskEnvelope
	if err := c.core.Do(ctx, op, http.MethodPost, u, body, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Task == nil {
		return nil, errors.New(op + ": empty response task")
	}
	return resp.Task, nil
}

// GetTask fetches a single task in an import session.
func (c *Client) GetTask(ctx context.Context, importID, taskID int64) (*Task, error) {
	const op = "Imports.GetTask"
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(importID, 10),
		"tasks", strconv.FormatInt(taskID, 10))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp taskEnvelope
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Task == nil {
		return nil, errors.New(op + ": empty response task")
	}
	return resp.Task, nil
}

// UpdateTask modifies a task — typically to change its `UpdateMode`
// (`CREATE` / `APPEND` / `REPLACE`).
func (c *Client) UpdateTask(ctx context.Context, importID, taskID int64, t *Task) error {
	const op = "Imports.UpdateTask"
	if t == nil {
		return errors.New(op + ": nil task")
	}
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(importID, 10),
		"tasks", strconv.FormatInt(taskID, 10))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, taskEnvelope{Task: t}, nil, nil)
}

// DeleteTask removes a task from an import session.
func (c *Client) DeleteTask(ctx context.Context, importID, taskID int64) error {
	const op = "Imports.DeleteTask"
	u, err := c.core.URL("rest", "imports", strconv.FormatInt(importID, 10),
		"tasks", strconv.FormatInt(taskID, 10))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
