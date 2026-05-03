package geoserver

import (
	"bytes"
	"context"
	"fmt"
)

// defaultUserGroupService is the GeoServer default user/group service name
// used when callers pass an empty service name.
const defaultUserGroupService = "default"

// User represents a GeoServer user record.
//
// Password is write-only — GeoServer never returns the password hash on GET,
// so this field is empty on values returned from list / fetch operations.
type User struct {
	Name     string `json:"userName"`
	Enabled  bool   `json:"enabled"`
	Password string `json:"password,omitempty"`
}

// Group represents a GeoServer user group.
type Group struct {
	Name string `json:"groupName"`
}

// SecurityService defines GeoServer security operations covering users,
// groups, and roles.
type SecurityService interface {
	// Users
	GetUsers(serviceName string) (users []User, err error)
	CreateUser(userName string, password string, serviceName string) (created bool, err error)
	DeleteUser(userName string, serviceName string) (deleted bool, err error)

	// Groups
	GetGroups(serviceName string) (groups []Group, err error)
	CreateGroup(groupName string, serviceName string) (created bool, err error)
	DeleteGroup(groupName string, serviceName string) (deleted bool, err error)

	// Roles
	GetRoles() (roles []string, err error)
	GetUserRoles(userName string) (roles []string, err error)
	CreateRole(roleName string) (created bool, err error)
	DeleteRole(roleName string) (deleted bool, err error)

	// User <-> Role association
	AddUserRole(roleName string, userName string) (added bool, err error)
	DeleteUserRole(roleName string, userName string) (deleted bool, err error)
}

// SecurityServiceWithContext is the context-aware sibling of [SecurityService].
type SecurityServiceWithContext interface {
	GetUsersContext(ctx context.Context, serviceName string) (users []User, err error)
	CreateUserContext(ctx context.Context, userName string, password string, serviceName string) (created bool, err error)
	DeleteUserContext(ctx context.Context, userName string, serviceName string) (deleted bool, err error)

	GetGroupsContext(ctx context.Context, serviceName string) (groups []Group, err error)
	CreateGroupContext(ctx context.Context, groupName string, serviceName string) (created bool, err error)
	DeleteGroupContext(ctx context.Context, groupName string, serviceName string) (deleted bool, err error)

	GetRolesContext(ctx context.Context) (roles []string, err error)
	GetUserRolesContext(ctx context.Context, userName string) (roles []string, err error)
	CreateRoleContext(ctx context.Context, roleName string) (created bool, err error)
	DeleteRoleContext(ctx context.Context, roleName string) (deleted bool, err error)

	AddUserRoleContext(ctx context.Context, roleName string, userName string) (added bool, err error)
	DeleteUserRoleContext(ctx context.Context, roleName string, userName string) (deleted bool, err error)
}

// resolveService returns the user-group service name to address, defaulting
// to "default" when serviceName is empty. GeoServer's default service is
// "default" out of the box.
func resolveService(serviceName string) string {
	if serviceName == "" {
		return defaultUserGroupService
	}
	return serviceName
}

// usersURL builds /rest/security/usergroup/service/{service}/users[/{user}].
func (g *GeoServer) usersURL(serviceName string, extra ...string) string {
	parts := make([]string, 0, 5+len(extra))
	parts = append(parts, "rest", "security", "usergroup", "service", resolveService(serviceName))
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// groupsURL builds /rest/security/usergroup/service/{service}/groups[...].
func (g *GeoServer) groupsURL(serviceName string, extra ...string) string {
	parts := make([]string, 0, 5+len(extra))
	parts = append(parts, "rest", "security", "usergroup", "service", resolveService(serviceName))
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// rolesURL builds /rest/security/roles[/...].
func (g *GeoServer) rolesURL(extra ...string) string {
	parts := make([]string, 0, 3+len(extra))
	parts = append(parts, "rest", "security", "roles")
	parts = append(parts, extra...)
	return g.ParseURL(parts...)
}

// ---- Users ---------------------------------------------------------------

// GetUsers returns users for the named user-group service using context.Background.
// An empty serviceName resolves to "default".
func (g *GeoServer) GetUsers(serviceName string) (users []User, err error) {
	return g.GetUsersContext(context.Background(), serviceName)
}

// GetUsersContext is the context-aware variant of [GeoServer.GetUsers].
func (g *GeoServer) GetUsersContext(ctx context.Context, serviceName string) (users []User, err error) {
	targetURL := g.usersURL(serviceName, "users")
	httpRequest := HTTPRequest{Method: getMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		return nil, g.GetError(responseCode, response)
	}
	var body struct {
		Users []User `json:"users"`
	}
	if err = g.DeSerializeJSON(response, &body); err != nil {
		return nil, fmt.Errorf("GetUsers: decode: %w", err)
	}
	return body.Users, nil
}

// CreateUser creates a user under the named service using context.Background.
// An empty serviceName resolves to "default".
func (g *GeoServer) CreateUser(userName string, password string, serviceName string) (created bool, err error) {
	return g.CreateUserContext(context.Background(), userName, password, serviceName)
}

// CreateUserContext is the context-aware variant of [GeoServer.CreateUser].
func (g *GeoServer) CreateUserContext(ctx context.Context, userName string, password string, serviceName string) (created bool, err error) {
	targetURL := g.usersURL(serviceName, "users")
	body := struct {
		User User `json:"user"`
	}{User{Name: userName, Enabled: true, Password: password}}
	data, serErr := g.SerializeStruct(body)
	if serErr != nil {
		return false, fmt.Errorf("CreateUser: serialize: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(data),
		DataType: jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteUser deletes a user from the named service using context.Background.
func (g *GeoServer) DeleteUser(userName string, serviceName string) (deleted bool, err error) {
	return g.DeleteUserContext(context.Background(), userName, serviceName)
}

// DeleteUserContext is the context-aware variant of [GeoServer.DeleteUser].
func (g *GeoServer) DeleteUserContext(ctx context.Context, userName string, serviceName string) (deleted bool, err error) {
	targetURL := g.usersURL(serviceName, "user", userName)
	httpRequest := HTTPRequest{Method: deleteMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// ---- Groups --------------------------------------------------------------

// GetGroups returns groups for the named user-group service using context.Background.
func (g *GeoServer) GetGroups(serviceName string) (groups []Group, err error) {
	return g.GetGroupsContext(context.Background(), serviceName)
}

// GetGroupsContext is the context-aware variant of [GeoServer.GetGroups].
//
// Decodes both `{"groups": [...]}` (GeoServer 2.28+) and `{"groupNames": [...]}`
// (older 2.x). Each entry is a bare group name string; we normalize to [Group].
func (g *GeoServer) GetGroupsContext(ctx context.Context, serviceName string) (groups []Group, err error) {
	targetURL := g.groupsURL(serviceName, "groups")
	httpRequest := HTTPRequest{Method: getMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		return nil, g.GetError(responseCode, response)
	}
	var body struct {
		Groups     []string `json:"groups,omitempty"`
		GroupNames []string `json:"groupNames,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &body); err != nil {
		return nil, fmt.Errorf("GetGroups: decode: %w", err)
	}
	names := body.Groups
	if len(names) == 0 {
		names = body.GroupNames
	}
	groups = make([]Group, 0, len(names))
	for _, name := range names {
		groups = append(groups, Group{Name: name})
	}
	return groups, nil
}

// CreateGroup creates a group under the named service using context.Background.
func (g *GeoServer) CreateGroup(groupName string, serviceName string) (created bool, err error) {
	return g.CreateGroupContext(context.Background(), groupName, serviceName)
}

// CreateGroupContext is the context-aware variant of [GeoServer.CreateGroup].
func (g *GeoServer) CreateGroupContext(ctx context.Context, groupName string, serviceName string) (created bool, err error) {
	targetURL := g.groupsURL(serviceName, "group", groupName)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		DataType: jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteGroup deletes a group from the named service using context.Background.
func (g *GeoServer) DeleteGroup(groupName string, serviceName string) (deleted bool, err error) {
	return g.DeleteGroupContext(context.Background(), groupName, serviceName)
}

// DeleteGroupContext is the context-aware variant of [GeoServer.DeleteGroup].
func (g *GeoServer) DeleteGroupContext(ctx context.Context, groupName string, serviceName string) (deleted bool, err error) {
	targetURL := g.groupsURL(serviceName, "group", groupName)
	httpRequest := HTTPRequest{Method: deleteMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// ---- Roles ---------------------------------------------------------------

// GetRoles returns all role names defined in GeoServer using context.Background.
func (g *GeoServer) GetRoles() (roles []string, err error) {
	return g.GetRolesContext(context.Background())
}

// GetRolesContext is the context-aware variant of [GeoServer.GetRoles].
//
// Decodes both `{"roles":[...]}` (GeoServer 2.28+) and `{"roleNames":[...]}`
// (older 2.x) response shapes; whichever is non-empty wins.
func (g *GeoServer) GetRolesContext(ctx context.Context) (roles []string, err error) {
	targetURL := g.rolesURL()
	httpRequest := HTTPRequest{Method: getMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		return nil, g.GetError(responseCode, response)
	}
	var body struct {
		Roles     []string `json:"roles,omitempty"`
		RoleNames []string `json:"roleNames,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &body); err != nil {
		return nil, fmt.Errorf("GetRoles: decode: %w", err)
	}
	if len(body.Roles) > 0 {
		return body.Roles, nil
	}
	return body.RoleNames, nil
}

// GetUserRoles returns the role names assigned to userName using context.Background.
//
// GeoServer returns an empty role list (not 404) for users that do not exist;
// callers wanting to validate user existence should use [GeoServer.GetUsers].
func (g *GeoServer) GetUserRoles(userName string) (roles []string, err error) {
	return g.GetUserRolesContext(context.Background(), userName)
}

// GetUserRolesContext is the context-aware variant of [GeoServer.GetUserRoles].
//
// Decodes both `{"roles":[...]}` (GeoServer 2.28+) and `{"roleNames":[...]}`
// (older 2.x) response shapes.
func (g *GeoServer) GetUserRolesContext(ctx context.Context, userName string) (roles []string, err error) {
	targetURL := g.rolesURL("user", userName)
	httpRequest := HTTPRequest{Method: getMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		return nil, g.GetError(responseCode, response)
	}
	var body struct {
		Roles     []string `json:"roles,omitempty"`
		RoleNames []string `json:"roleNames,omitempty"`
	}
	if err = g.DeSerializeJSON(response, &body); err != nil {
		return nil, fmt.Errorf("GetUserRoles: decode: %w", err)
	}
	if len(body.Roles) > 0 {
		return body.Roles, nil
	}
	return body.RoleNames, nil
}

// CreateRole creates a role using context.Background.
func (g *GeoServer) CreateRole(roleName string) (created bool, err error) {
	return g.CreateRoleContext(context.Background(), roleName)
}

// CreateRoleContext is the context-aware variant of [GeoServer.CreateRole].
func (g *GeoServer) CreateRoleContext(ctx context.Context, roleName string) (created bool, err error) {
	targetURL := g.rolesURL("role", roleName)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		DataType: jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusCreated {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteRole deletes a role using context.Background.
func (g *GeoServer) DeleteRole(roleName string) (deleted bool, err error) {
	return g.DeleteRoleContext(context.Background(), roleName)
}

// DeleteRoleContext is the context-aware variant of [GeoServer.DeleteRole].
func (g *GeoServer) DeleteRoleContext(ctx context.Context, roleName string) (deleted bool, err error) {
	targetURL := g.rolesURL("role", roleName)
	httpRequest := HTTPRequest{Method: deleteMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// ---- User <-> Role association ------------------------------------------

// AddUserRole associates roleName with userName using context.Background.
//
// GeoServer returns 200 OK (not 201 Created) for role assignments.
// Re-assigning a role that is already attached is idempotent — GeoServer
// returns 200 OK without an error.
func (g *GeoServer) AddUserRole(roleName string, userName string) (added bool, err error) {
	return g.AddUserRoleContext(context.Background(), roleName, userName)
}

// AddUserRoleContext is the context-aware variant of [GeoServer.AddUserRole].
func (g *GeoServer) AddUserRoleContext(ctx context.Context, roleName string, userName string) (added bool, err error) {
	targetURL := g.rolesURL("role", roleName, "user", userName)
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		DataType: jsonType,
		URL:      targetURL,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteUserRole disassociates roleName from userName using context.Background.
//
// GeoServer returns 200 OK whether or not the user actually had the role
// assigned, as long as both the role and user exist. If either is missing,
// the response is non-2xx and a typed error is returned.
func (g *GeoServer) DeleteUserRole(roleName string, userName string) (deleted bool, err error) {
	return g.DeleteUserRoleContext(context.Background(), roleName, userName)
}

// DeleteUserRoleContext is the context-aware variant of [GeoServer.DeleteUserRole].
func (g *GeoServer) DeleteUserRoleContext(ctx context.Context, roleName string, userName string) (deleted bool, err error) {
	targetURL := g.rolesURL("role", roleName, "user", userName)
	httpRequest := HTTPRequest{Method: deleteMethod, Accept: jsonType, URL: targetURL}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}
