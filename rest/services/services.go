package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
// Defined here as an interface so this subpackage doesn't import the
// root package (which would create an import cycle).
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
}

// Client is the v2 services entry-point sub-client. Use the per-OWS
// accessors to reach the typed clients:
//
//	c.Services.WMS().Get(ctx)
//	c.Services.WFS().InWorkspace("topp").Update(ctx, settings)
//	c.Services.WCS().InWorkspace("nurc").Delete(ctx)
//
// Construct via the parent [*geoserver.Client]; do not call [New]
// directly outside the root package's wiring.
type Client struct {
	core Core
}

// New constructs the entry-point client. Used by the root
// [*geoserver.Client] wiring; library users access the same
// instance via `c.Services`.
func New(core Core) *Client {
	return &Client{core: core}
}

// WMS returns the global-scope WMS settings client.
func (c *Client) WMS() *WMSClient { return &WMSClient{core: c.core} }

// WFS returns the global-scope WFS settings client.
func (c *Client) WFS() *WFSClient { return &WFSClient{core: c.core} }

// WCS returns the global-scope WCS settings client.
func (c *Client) WCS() *WCSClient { return &WCSClient{core: c.core} }

// WMTS returns the global-scope WMTS settings client. Note: the
// upstream schema documents no unique fields for WMTS beyond
// [ServiceInfo].
func (c *Client) WMTS() *WMTSClient { return &WMTSClient{core: c.core} }

// urlParts builds the URL path parts for a service-settings
// endpoint. workspace is empty for the global form.
func urlParts(slug, workspace string) []string {
	parts := []string{"rest", "services", slug}
	if workspace != "" {
		parts = append(parts, "workspaces", workspace)
	}
	parts = append(parts, "settings")
	return parts
}

// ----- WMS -----

// WMSClient is the WMS settings client.
type WMSClient struct{ core Core }

// InWorkspace returns a workspace-scoped WMS settings client. The
// workspace-scoped client has a Delete method that removes the
// override and falls back to the global config.
func (c *WMSClient) InWorkspace(workspace string) *WMSWorkspaceClient {
	return &WMSWorkspaceClient{core: c.core, workspace: workspace}
}

// Get fetches the global WMS settings document.
func (c *WMSClient) Get(ctx context.Context) (*WMSSettings, error) {
	return getWMS(ctx, c.core, "WMS.Get", "")
}

// Update writes the global WMS settings via PUT.
func (c *WMSClient) Update(ctx context.Context, s *WMSSettings) error {
	return putWMS(ctx, c.core, "WMS.Update", "", s)
}

// WMSWorkspaceClient is the per-workspace WMS settings client.
type WMSWorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WMSWorkspaceClient) Workspace() string { return c.workspace }

// Get fetches the per-workspace WMS settings override. Returns a
// *APIError wrapping ErrNotFound if no override is configured.
func (c *WMSWorkspaceClient) Get(ctx context.Context) (*WMSSettings, error) {
	if c.workspace == "" {
		return nil, errors.New("WMS.InWorkspace.Get: empty workspace name")
	}
	return getWMS(ctx, c.core, "WMS.InWorkspace.Get", c.workspace)
}

// Update writes the per-workspace WMS settings override.
func (c *WMSWorkspaceClient) Update(ctx context.Context, s *WMSSettings) error {
	if c.workspace == "" {
		return errors.New("WMS.InWorkspace.Update: empty workspace name")
	}
	return putWMS(ctx, c.core, "WMS.InWorkspace.Update", c.workspace, s)
}

// Delete removes the per-workspace WMS settings override; the
// workspace falls back to the global configuration.
func (c *WMSWorkspaceClient) Delete(ctx context.Context) error {
	if c.workspace == "" {
		return errors.New("WMS.InWorkspace.Delete: empty workspace name")
	}
	return deleteService(ctx, c.core, "WMS.InWorkspace.Delete", "wms", c.workspace)
}

// ----- WFS -----

// WFSClient is the WFS settings client.
type WFSClient struct{ core Core }

// InWorkspace returns a workspace-scoped WFS settings client.
func (c *WFSClient) InWorkspace(workspace string) *WFSWorkspaceClient {
	return &WFSWorkspaceClient{core: c.core, workspace: workspace}
}

// Get fetches the global WFS settings document.
func (c *WFSClient) Get(ctx context.Context) (*WFSSettings, error) {
	return getWFS(ctx, c.core, "WFS.Get", "")
}

// Update writes the global WFS settings via PUT.
func (c *WFSClient) Update(ctx context.Context, s *WFSSettings) error {
	return putWFS(ctx, c.core, "WFS.Update", "", s)
}

// WFSWorkspaceClient is the per-workspace WFS settings client.
type WFSWorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WFSWorkspaceClient) Workspace() string { return c.workspace }

// Get fetches the per-workspace WFS settings override.
func (c *WFSWorkspaceClient) Get(ctx context.Context) (*WFSSettings, error) {
	if c.workspace == "" {
		return nil, errors.New("WFS.InWorkspace.Get: empty workspace name")
	}
	return getWFS(ctx, c.core, "WFS.InWorkspace.Get", c.workspace)
}

// Update writes the per-workspace WFS settings override.
func (c *WFSWorkspaceClient) Update(ctx context.Context, s *WFSSettings) error {
	if c.workspace == "" {
		return errors.New("WFS.InWorkspace.Update: empty workspace name")
	}
	return putWFS(ctx, c.core, "WFS.InWorkspace.Update", c.workspace, s)
}

// Delete removes the per-workspace WFS settings override.
func (c *WFSWorkspaceClient) Delete(ctx context.Context) error {
	if c.workspace == "" {
		return errors.New("WFS.InWorkspace.Delete: empty workspace name")
	}
	return deleteService(ctx, c.core, "WFS.InWorkspace.Delete", "wfs", c.workspace)
}

// ----- WCS -----

// WCSClient is the WCS settings client.
type WCSClient struct{ core Core }

// InWorkspace returns a workspace-scoped WCS settings client.
func (c *WCSClient) InWorkspace(workspace string) *WCSWorkspaceClient {
	return &WCSWorkspaceClient{core: c.core, workspace: workspace}
}

// Get fetches the global WCS settings document.
func (c *WCSClient) Get(ctx context.Context) (*WCSSettings, error) {
	return getWCS(ctx, c.core, "WCS.Get", "")
}

// Update writes the global WCS settings via PUT.
func (c *WCSClient) Update(ctx context.Context, s *WCSSettings) error {
	return putWCS(ctx, c.core, "WCS.Update", "", s)
}

// WCSWorkspaceClient is the per-workspace WCS settings client.
type WCSWorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WCSWorkspaceClient) Workspace() string { return c.workspace }

// Get fetches the per-workspace WCS settings override.
func (c *WCSWorkspaceClient) Get(ctx context.Context) (*WCSSettings, error) {
	if c.workspace == "" {
		return nil, errors.New("WCS.InWorkspace.Get: empty workspace name")
	}
	return getWCS(ctx, c.core, "WCS.InWorkspace.Get", c.workspace)
}

// Update writes the per-workspace WCS settings override.
func (c *WCSWorkspaceClient) Update(ctx context.Context, s *WCSSettings) error {
	if c.workspace == "" {
		return errors.New("WCS.InWorkspace.Update: empty workspace name")
	}
	return putWCS(ctx, c.core, "WCS.InWorkspace.Update", c.workspace, s)
}

// Delete removes the per-workspace WCS settings override.
func (c *WCSWorkspaceClient) Delete(ctx context.Context) error {
	if c.workspace == "" {
		return errors.New("WCS.InWorkspace.Delete: empty workspace name")
	}
	return deleteService(ctx, c.core, "WCS.InWorkspace.Delete", "wcs", c.workspace)
}

// ----- WMTS -----

// WMTSClient is the WMTS settings client.
type WMTSClient struct{ core Core }

// InWorkspace returns a workspace-scoped WMTS settings client.
func (c *WMTSClient) InWorkspace(workspace string) *WMTSWorkspaceClient {
	return &WMTSWorkspaceClient{core: c.core, workspace: workspace}
}

// Get fetches the global WMTS settings document.
func (c *WMTSClient) Get(ctx context.Context) (*WMTSSettings, error) {
	return getWMTS(ctx, c.core, "WMTS.Get", "")
}

// Update writes the global WMTS settings via PUT.
func (c *WMTSClient) Update(ctx context.Context, s *WMTSSettings) error {
	return putWMTS(ctx, c.core, "WMTS.Update", "", s)
}

// WMTSWorkspaceClient is the per-workspace WMTS settings client.
type WMTSWorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WMTSWorkspaceClient) Workspace() string { return c.workspace }

// Get fetches the per-workspace WMTS settings override.
func (c *WMTSWorkspaceClient) Get(ctx context.Context) (*WMTSSettings, error) {
	if c.workspace == "" {
		return nil, errors.New("WMTS.InWorkspace.Get: empty workspace name")
	}
	return getWMTS(ctx, c.core, "WMTS.InWorkspace.Get", c.workspace)
}

// Update writes the per-workspace WMTS settings override.
func (c *WMTSWorkspaceClient) Update(ctx context.Context, s *WMTSSettings) error {
	if c.workspace == "" {
		return errors.New("WMTS.InWorkspace.Update: empty workspace name")
	}
	return putWMTS(ctx, c.core, "WMTS.InWorkspace.Update", c.workspace, s)
}

// Delete removes the per-workspace WMTS settings override.
func (c *WMTSWorkspaceClient) Delete(ctx context.Context) error {
	if c.workspace == "" {
		return errors.New("WMTS.InWorkspace.Delete: empty workspace name")
	}
	return deleteService(ctx, c.core, "WMTS.InWorkspace.Delete", "wmts", c.workspace)
}

// ----- internal helpers (per-service Get/Update; shared Delete) -----
//
// Each service's helpers know the slug, the envelope wrapper, and
// the concrete Settings type. The per-service WMSClient/WFSClient/...
// methods route through these.

func getWMS(ctx context.Context, core Core, op, ws string) (*WMSSettings, error) {
	u, err := core.URL(urlParts("wms", ws)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var env wmsEnvelope
	if err := core.Do(ctx, op, http.MethodGet, u, nil, nil, &env); err != nil {
		return nil, err
	}
	if env.WMS == nil {
		env.WMS = &WMSSettings{}
	}
	return env.WMS, nil
}

func putWMS(ctx context.Context, core Core, op, ws string, s *WMSSettings) error {
	if s == nil {
		return errors.New(op + ": nil settings")
	}
	u, err := core.URL(urlParts("wms", ws)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return core.Do(ctx, op, http.MethodPut, u, wmsEnvelope{WMS: s}, nil, nil)
}

func getWFS(ctx context.Context, core Core, op, ws string) (*WFSSettings, error) {
	u, err := core.URL(urlParts("wfs", ws)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var env wfsEnvelope
	if err := core.Do(ctx, op, http.MethodGet, u, nil, nil, &env); err != nil {
		return nil, err
	}
	if env.WFS == nil {
		env.WFS = &WFSSettings{}
	}
	return env.WFS, nil
}

func putWFS(ctx context.Context, core Core, op, ws string, s *WFSSettings) error {
	if s == nil {
		return errors.New(op + ": nil settings")
	}
	u, err := core.URL(urlParts("wfs", ws)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return core.Do(ctx, op, http.MethodPut, u, wfsEnvelope{WFS: s}, nil, nil)
}

func getWCS(ctx context.Context, core Core, op, ws string) (*WCSSettings, error) {
	u, err := core.URL(urlParts("wcs", ws)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var env wcsEnvelope
	if err := core.Do(ctx, op, http.MethodGet, u, nil, nil, &env); err != nil {
		return nil, err
	}
	if env.WCS == nil {
		env.WCS = &WCSSettings{}
	}
	return env.WCS, nil
}

func putWCS(ctx context.Context, core Core, op, ws string, s *WCSSettings) error {
	if s == nil {
		return errors.New(op + ": nil settings")
	}
	u, err := core.URL(urlParts("wcs", ws)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return core.Do(ctx, op, http.MethodPut, u, wcsEnvelope{WCS: s}, nil, nil)
}

func getWMTS(ctx context.Context, core Core, op, ws string) (*WMTSSettings, error) {
	u, err := core.URL(urlParts("wmts", ws)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var env wmtsEnvelope
	if err := core.Do(ctx, op, http.MethodGet, u, nil, nil, &env); err != nil {
		return nil, err
	}
	if env.WMTS == nil {
		env.WMTS = &WMTSSettings{}
	}
	return env.WMTS, nil
}

func putWMTS(ctx context.Context, core Core, op, ws string, s *WMTSSettings) error {
	if s == nil {
		return errors.New(op + ": nil settings")
	}
	u, err := core.URL(urlParts("wmts", ws)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return core.Do(ctx, op, http.MethodPut, u, wmtsEnvelope{WMTS: s}, nil, nil)
}

// deleteService is shared because DELETE is identical across all
// four services (no body, no per-service envelope).
func deleteService(ctx context.Context, core Core, op, slug, ws string) error {
	u, err := core.URL(urlParts(slug, ws)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
