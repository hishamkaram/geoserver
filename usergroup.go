package geoserver

type User struct {
	Name     string `json:"userName"`
	Enabled  bool   `json:"enabled"`
	Password string `json:"password,omitempty"`
}

type rolesResponse struct {
	Roles []string `json:"roleNames"`
}

// GetUsers returns all users for service, if service is empty, returns users for default service
// err is an error if error occurred else err is nil
func (g *GeoServer) GetUsers(service string) (users []User, err error) {
	if service == "" {
		service = "default"
	}

	var usersResponse struct {
		Users []User `json:"users"`
	}

	targetURL := g.ParseURL("rest", "security", "usergroup", "service", service, "users")

	err = g.requestResource(targetURL, &usersResponse)

	return usersResponse.Users, err
}

// GetGroups returns all groups for service, if service is empty, returns groups for default service
// err is an error if error occurred else err is nil
func (g *GeoServer) GetGroups(service string) (users []User, err error) {
	if service == "" {
		service = "default"
	}

	var groupsResponse struct {
		Groups []User `json:"groupNames"`
	}

	targetURL := g.ParseURL("rest", "security", "usergroup", "service", service, "groups")

	err = g.requestResource(targetURL, &groupsResponse)

	return groupsResponse.Groups, err
}

// GetRoles returns all roles
// err is an error if error occurred else err is nil
func (g *GeoServer) GetRoles() (roles []string, err error) {
	targetURL := g.ParseURL("rest", "security", "roles")

	var resp rolesResponse
	err = g.requestResource(targetURL, &resp)

	return resp.Roles, err
}

// GetUserRoles returns all roles for user
// err is an error if error occurred else err is nil
func (g *GeoServer) GetUserRoles(user string) (roles []string, err error) {
	targetURL := g.ParseURL("rest", "security", "roles", "user", user)

	var resp rolesResponse
	err = g.requestResource(targetURL, &resp)

	return resp.Roles, err
}

// CreateUser creates user with name userName and password for service serviceName, if service is empty creates user for default service
// returns true/false if created or not, err is an error if error occurred else err is nil
func (g *GeoServer) CreateUser(userName string, password string, serviceName string) (created bool, err error) {
	if serviceName == "" {
		serviceName = "default"
	}
	targetURL := g.ParseURL("rest", "security", "usergroup", "service", serviceName, "users")

	createUserRequest := struct {
		User User `json:"user"`
	}{User{userName, true, password}}

	return g.createEntity(targetURL, createUserRequest, nil)
}

// DeleteUser deletes the user with name userName for service serviceName, if service is empty creates user for default service
// returns true/false if deleted or not, err is an error if error occurred else err is nil
func (g *GeoServer) DeleteUser(userName string, serviceName string) (done bool, err error) {
	if serviceName == "" {
		serviceName = "default"
	}
	targetURL := g.ParseURL("rest", "security", "usergroup", "service", serviceName, "user", userName)
	return g.deleteEntity(targetURL)
}

// CreateGroup creates group with name groupName for service serviceName, if service is empty creates user for default service
// returns true/false if created or not, err is an error if error occurred else err is nil
func (g *GeoServer) CreateGroup(groupName string, serviceName string) (created bool, err error) {
	if serviceName == "" {
		serviceName = "default"
	}
	targetURL := g.ParseURL("rest", "security", "usergroup", "service", serviceName, "group", groupName)

	return g.createEntity(targetURL, nil, nil)
}

// DeleteGroup deletes the group with name groupName
// returns true/false if deleted or not, err is an error if error occurred else err is nil
func (g *GeoServer) DeleteGroup(groupName string, serviceName string) (done bool, err error) {
	if serviceName == "" {
		serviceName = "default"
	}
	targetURL := g.ParseURL("rest", "security", "usergroup", "service", serviceName, "group", groupName)
	return g.deleteEntity(targetURL)
}

// CreateRole creates role with name roleName
// returns true/false if created or not, err is an error if error occurred else err is nil
func (g *GeoServer) CreateRole(roleName string) (created bool, err error) {
	targetURL := g.ParseURL("rest", "security", "roles", "role", roleName)

	return g.createEntity(targetURL, nil, nil)
}

// DeleteRole deletes the role with name roleName
// returns true/false if deleted or not, err is an error if error occurred else err is nil
func (g *GeoServer) DeleteRole(roleName string) (done bool, err error) {
	targetURL := g.ParseURL("rest", "security", "roles", "role", roleName)
	return g.deleteEntity(targetURL)
}

// AddUserRole adds (associates) role with name roleName to the user with name userName
// returns true/false if added or not, err is an error if error occurred else err is nil
func (g *GeoServer) AddUserRole(roleName string, userName string) (created bool, err error) {
	targetURL := g.ParseURL("rest", "security", "roles", "role", roleName, "user", userName)

	return g.createEntity(targetURL, nil, func(statusCode int, response []byte) error {
		if statusCode != statusOk {
			g.logger.Error(string(response))
			return g.GetError(statusCode, response)
		}
		return nil
	})

}

// DeleteUserRole deletes (disassociates) role with name roleName from the user with name userName
// returns true/false if deleted or not, err is an error if error occurred else err is nil
func (g *GeoServer) DeleteUserRole(roleName string, userName string) (done bool, err error) {
	targetURL := g.ParseURL("rest", "security", "roles", "role", roleName, "user", userName)
	return g.deleteEntity(targetURL)
}
