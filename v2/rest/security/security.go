package security

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
	// SynthesizeError surfaces a package-sentinel error (via the
	// parent's *APIError) for wire responses that are 2xx but
	// semantically failures — e.g. auth filters' `{"null":""}`
	// "not found" wire shape.
	SynthesizeError(op, method, requestURL string, statusCode int, bodyHint string) error
}

// Client is the v2 security sub-client. It carries six nested
// surfaces:
//
//	c.Security.Users()                       // default user/group service
//	c.Security.UsersInService("custom-jdbc") // custom service
//	c.Security.Groups()
//	c.Security.GroupsInService("custom-jdbc")
//	c.Security.Roles                         // always global, no service scope
//	c.Security.AuthProviders                 // /security/authproviders
//	c.Security.AuthFilters                   // /security/authfilters
//	c.Security.FilterChains                  // /security/filterchain
type Client struct {
	core Core

	// Roles is the entry point for role operations and user-role
	// assignment. Roles are global — not scoped to a user/group
	// service — so this is a single client, not a method.
	Roles *RolesClient

	// AuthProviders is the entry point for authentication-provider
	// CRUD plus active-order management at /security/authproviders.
	// Providers are how GeoServer delegates authentication to backends
	// (UsernamePassword, LDAP, OAuth/OIDC, header-auth, etc.).
	AuthProviders *AuthProvidersClient

	// AuthFilters is the entry point for authentication-filter CRUD at
	// /security/authfilters. Filters are individual auth steps
	// (anonymous, basic, form, rememberme, oidc-test, …) that are
	// composed into [FilterChainsClient] entries.
	AuthFilters *AuthFiltersClient

	// FilterChains is the entry point for security filter-chain CRUD
	// plus chain-order management at /security/filterchain. A filter
	// chain binds a URL pattern (e.g. "/web/**") to an ordered list
	// of [AuthFiltersClient] filter names.
	FilterChains *FilterChainsClient
}

// New constructs the security sub-client.
func New(core Core) *Client {
	return &Client{
		core:          core,
		Roles:         &RolesClient{core: core},
		AuthProviders: &AuthProvidersClient{core: core},
		AuthFilters:   &AuthFiltersClient{core: core},
		FilterChains:  &FilterChainsClient{core: core},
	}
}

// Users returns a users sub-client scoped to the default user/group
// service ("default"). For a custom service use
// [Client.UsersInService].
func (c *Client) Users() *UsersClient {
	return c.UsersInService("")
}

// UsersInService returns a users sub-client scoped to the named
// user/group service. Empty serviceName resolves to "default".
func (c *Client) UsersInService(serviceName string) *UsersClient {
	return &UsersClient{core: c.core, service: resolveService(serviceName)}
}

// Groups returns a groups sub-client scoped to the default user/group
// service. For a custom service use [Client.GroupsInService].
func (c *Client) Groups() *GroupsClient {
	return c.GroupsInService("")
}

// GroupsInService returns a groups sub-client scoped to the named
// user/group service. Empty serviceName resolves to "default".
func (c *Client) GroupsInService(serviceName string) *GroupsClient {
	return &GroupsClient{core: c.core, service: resolveService(serviceName)}
}

// ---- Users ----------------------------------------------------------------

// UsersClient is the user/group-service-scoped users client.
type UsersClient struct {
	core    Core
	service string
}

// Service returns the user/group service name this client targets.
func (c *UsersClient) Service() string { return c.service }

// List returns every user in the scoped user/group service.
func (c *UsersClient) List(ctx context.Context, _ ListOptions) ([]User, error) {
	const op = "Security.Users.List"
	u, err := c.core.URL("rest", "security", "usergroup", "service", c.service, "users")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp userListResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Users, nil
}

// Create registers a new user in the scoped service. user.Name and
// user.Password are required; user.Enabled defaults to true if the
// caller doesn't set it (use a deliberately-disabled user via the
// modify path once GeoServer supports an Update method).
func (c *UsersClient) Create(ctx context.Context, user *User) error {
	const op = "Security.Users.Create"
	if user == nil {
		return errors.New(op + ": nil user")
	}
	if user.Name == "" {
		return errors.New(op + ": empty user Name")
	}
	if user.Password == "" {
		return errors.New(op + ": empty user Password")
	}
	u, err := c.core.URL("rest", "security", "usergroup", "service", c.service, "users")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := userCreateRequest{User: *user}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Delete removes a user from the scoped service.
func (c *UsersClient) Delete(ctx context.Context, name string) error {
	const op = "Security.Users.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "usergroup", "service", c.service, "user", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// ---- Groups ---------------------------------------------------------------

// GroupsClient is the user/group-service-scoped groups client.
type GroupsClient struct {
	core    Core
	service string
}

// Service returns the user/group service name this client targets.
func (c *GroupsClient) Service() string { return c.service }

// List returns every group in the scoped user/group service.
//
// Decodes both `{"groups":[...]}` (GeoServer 2.28+) and
// `{"groupNames":[...]}` (older 2.x) response shapes. Each entry is a
// bare name string; the wrapper [Group] type is for forward-
// compatibility.
func (c *GroupsClient) List(ctx context.Context, _ ListOptions) ([]Group, error) {
	const op = "Security.Groups.List"
	u, err := c.core.URL("rest", "security", "usergroup", "service", c.service, "groups")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp nameListResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	names := resp.names()
	groups := make([]Group, 0, len(names))
	for _, name := range names {
		groups = append(groups, Group{Name: name})
	}
	return groups, nil
}

// Create registers a new group in the scoped service. The wire path
// addresses the named group directly (POST .../group/{name}); no body
// is sent.
func (c *GroupsClient) Create(ctx context.Context, name string) error {
	const op = "Security.Groups.Create"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "usergroup", "service", c.service, "group", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}

// Delete removes a group from the scoped service.
func (c *GroupsClient) Delete(ctx context.Context, name string) error {
	const op = "Security.Groups.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "usergroup", "service", c.service, "group", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// ---- Roles ----------------------------------------------------------------

// RolesClient covers role CRUD plus user-role assignment. Roles are
// global — not scoped to a user/group service.
type RolesClient struct {
	core Core
}

// List returns every role name defined in GeoServer.
//
// Decodes both `{"roles":[...]}` (GeoServer 2.28+) and
// `{"roleNames":[...]}` (older 2.x) response shapes.
func (c *RolesClient) List(ctx context.Context, _ ListOptions) ([]string, error) {
	const op = "Security.Roles.List"
	u, err := c.core.URL("rest", "security", "roles")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp nameListResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.names(), nil
}

// Create registers a new role.
func (c *RolesClient) Create(ctx context.Context, name string) error {
	const op = "Security.Roles.Create"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "roles", "role", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}

// Delete removes a role.
func (c *RolesClient) Delete(ctx context.Context, name string) error {
	const op = "Security.Roles.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "roles", "role", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// ForUser returns the roles assigned to userName.
//
// GeoServer returns an empty role list (200 OK, not 404) for users
// that don't exist; validate user existence separately if needed.
//
// Same `{"roles":...}` / `{"roleNames":...}` cross-version handling
// as [RolesClient.List].
func (c *RolesClient) ForUser(ctx context.Context, userName string) ([]string, error) {
	const op = "Security.Roles.ForUser"
	if userName == "" {
		return nil, errors.New(op + ": empty userName")
	}
	u, err := c.core.URL("rest", "security", "roles", "user", userName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp nameListResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.names(), nil
}

// AssignToUser associates roleName with userName.
//
// GeoServer returns 200 OK (not 201) for assignment; re-assigning a
// role already attached is idempotent.
func (c *RolesClient) AssignToUser(ctx context.Context, roleName, userName string) error {
	const op = "Security.Roles.AssignToUser"
	if roleName == "" {
		return errors.New(op + ": empty roleName")
	}
	if userName == "" {
		return errors.New(op + ": empty userName")
	}
	u, err := c.core.URL("rest", "security", "roles", "role", roleName, "user", userName)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}

// UnassignFromUser disassociates roleName from userName.
//
// GeoServer returns 200 OK whether or not the user actually had the
// role attached, so long as both exist. If either doesn't exist, the
// response is non-2xx and a *APIError is returned.
func (c *RolesClient) UnassignFromUser(ctx context.Context, roleName, userName string) error {
	const op = "Security.Roles.UnassignFromUser"
	if roleName == "" {
		return errors.New(op + ": empty roleName")
	}
	if userName == "" {
		return errors.New(op + ": empty userName")
	}
	u, err := c.core.URL("rest", "security", "roles", "role", roleName, "user", userName)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
