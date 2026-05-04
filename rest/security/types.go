// Package security is the v2 sub-client for the GeoServer security
// subsystem — users, groups, roles, and user-role assignment under
// /rest/security.
//
// Users and groups are partitioned by user/group service (typically
// "default"); roles are global. Use [Client.Users] / [Client.Groups]
// for the default service or [Client.UsersInService] /
// [Client.GroupsInService] for a custom one. Roles live on
// [Client.Roles].
package security

// User represents a GeoServer user record.
//
// Password is write-only — GeoServer does not return the password
// hash on read paths, so this field is empty on List results.
type User struct {
	Name     string `json:"userName"`
	Enabled  bool   `json:"enabled"`
	Password string `json:"password,omitempty"`
}

// Group represents a GeoServer user group. The wire shape carries
// only the group name; v2 wraps it in a struct for forward
// compatibility (description / metadata fields could be added without
// a breaking change).
type Group struct {
	Name string `json:"groupName"`
}

// ListOptions controls listing behavior. Currently empty across
// users / groups / roles; reserved for future fields.
type ListOptions struct{}

// DefaultService is the GeoServer default user/group service name.
// Empty service names passed to [Client.UsersInService] /
// [Client.GroupsInService] resolve to this value.
const DefaultService = "default"

// resolveService returns the user-group service name to address,
// defaulting to [DefaultService] when serviceName is empty.
func resolveService(serviceName string) string {
	if serviceName == "" {
		return DefaultService
	}
	return serviceName
}

// userListResponse mirrors GeoServer's `{"users":[...]}`.
type userListResponse struct {
	Users []User `json:"users"`
}

// userCreateRequest mirrors GeoServer's create body shape.
type userCreateRequest struct {
	User User `json:"user"`
}

// nameListResponse handles GeoServer's two cross-version shapes for
// listing names: `{"roles":[]}` (2.28+) and `{"roleNames":[]}` (older
// 2.x), and the same `groups`/`groupNames` split. Whichever is
// non-empty wins.
type nameListResponse struct {
	Roles      []string `json:"roles,omitempty"`
	RoleNames  []string `json:"roleNames,omitempty"`
	Groups     []string `json:"groups,omitempty"`
	GroupNames []string `json:"groupNames,omitempty"`
}

func (r *nameListResponse) names() []string {
	switch {
	case len(r.Roles) > 0:
		return r.Roles
	case len(r.RoleNames) > 0:
		return r.RoleNames
	case len(r.Groups) > 0:
		return r.Groups
	case len(r.GroupNames) > 0:
		return r.GroupNames
	default:
		return nil
	}
}
