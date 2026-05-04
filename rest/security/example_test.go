package security_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/security"
)

// ExampleClient_Users returns the users sub-client for the default
// user/group service. Use [Client.UsersInService] for a custom
// service.
func ExampleClient_Users() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	users := c.Security.Users()
	for _, u := range mustList(users.List(context.Background(), security.ListOptions{})) {
		fmt.Printf("%s enabled=%v\n", u.Name, u.Enabled)
	}
}

// ExampleUsersClient_Create registers a new user in the default
// user/group service. The Password field is write-only — GeoServer
// won't return the hash on subsequent reads.
func ExampleUsersClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Security.Users().Create(context.Background(), &security.User{
		Name:     "alice",
		Password: "s3cret",
		Enabled:  true,
	})
}

// ExampleRolesClient_AssignToUser grants a role to a user. The role
// must already exist (create with [RolesClient.Create]).
func ExampleRolesClient_AssignToUser() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Security.Roles.Create(context.Background(), "EDITOR")
	_ = c.Security.Roles.AssignToUser(context.Background(), "EDITOR", "alice")
}

// ExampleRolesClient_ForUser returns every role assigned to a user.
// Useful for an authz audit pass.
func ExampleRolesClient_ForUser() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	roles, err := c.Security.Roles.ForUser(context.Background(), "alice")
	if err != nil {
		return
	}
	for _, r := range roles {
		fmt.Println(r)
	}
}

// mustList is an example helper: panics on List errors so the example
// stays focused on the successful path. Real code should match
// [errors.Is] against the package sentinels and recover gracefully.
func mustList[T any](xs []T, err error) []T {
	if err != nil {
		panic(err)
	}
	return xs
}
