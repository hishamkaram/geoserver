package geoserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity_GetUsers_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/security/usergroup/service/default/users", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"users":[{"userName":"admin","enabled":true},{"userName":"viewer","enabled":true}]}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	users, err := gs.GetUsersContext(context.Background(), "")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Name)
}

func TestSecurity_GetUsers_NamedService_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/service/customService/")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "service not found")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	_, err := gs.GetUsersContext(context.Background(), "customService")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSecurity_CreateUser_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/security/usergroup/service/default/users", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"userName":"alice"`)
		assert.Contains(t, string(body), `"password":"secret"`)
		assert.Contains(t, string(body), `"enabled":true`)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateUserContext(context.Background(), "alice", "secret", "")
	assert.NoError(t, err)
	assert.True(t, created)
}

func TestSecurity_CreateUser_409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "user already exists")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateUserContext(context.Background(), "alice", "secret", "")
	assert.False(t, created)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestSecurity_DeleteUser_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/service/default/user/alice"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteUserContext(context.Background(), "alice", "")
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestSecurity_GetGroups_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/security/usergroup/service/default/groups", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// GeoServer 2.28 uses "groups"; pre-2.28 used "groupNames".
		_, _ = io.WriteString(w, `{"groups":["editors","viewers"]}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	groups, err := gs.GetGroupsContext(context.Background(), "")
	assert.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Equal(t, "editors", groups[0].Name)
	assert.Equal(t, "viewers", groups[1].Name)
}

func TestSecurity_CreateGroup_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/service/default/group/editors"))
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateGroupContext(context.Background(), "editors", "")
	assert.NoError(t, err)
	assert.True(t, created)
}

func TestSecurity_DeleteGroup_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteGroupContext(context.Background(), "editors", "")
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestSecurity_GetRoles_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/security/roles", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// GeoServer 2.28 uses "roles"; pre-2.28 used "roleNames".
		_, _ = io.WriteString(w, `{"roles":["ROLE_ADMINISTRATOR","ROLE_VIEWER"]}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	roles, err := gs.GetRolesContext(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []string{"ROLE_ADMINISTRATOR", "ROLE_VIEWER"}, roles)
}

func TestSecurity_GetUserRoles_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/rest/security/roles/user/admin"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"roles":["ROLE_ADMINISTRATOR"]}`)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	roles, err := gs.GetUserRolesContext(context.Background(), "admin")
	assert.NoError(t, err)
	assert.Equal(t, []string{"ROLE_ADMINISTRATOR"}, roles)
}

func TestSecurity_CreateRole_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/rest/security/roles/role/ROLE_X"))
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	created, err := gs.CreateRoleContext(context.Background(), "ROLE_X")
	assert.NoError(t, err)
	assert.True(t, created)
}

func TestSecurity_DeleteRole_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "role missing")
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteRoleContext(context.Background(), "ROLE_GONE")
	assert.False(t, deleted)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSecurity_AddUserRole_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/rest/security/roles/role/ROLE_VIEWER/user/alice"))
		// GeoServer returns 200 (not 201) for role assignments.
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	added, err := gs.AddUserRoleContext(context.Background(), "ROLE_VIEWER", "alice")
	assert.NoError(t, err)
	assert.True(t, added)
}

func TestSecurity_DeleteUserRole_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	gs := newTestCatalog(srv)
	deleted, err := gs.DeleteUserRoleContext(context.Background(), "ROLE_VIEWER", "alice")
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestSecurity_ServiceImpl(t *testing.T) {
	// Compile-time + runtime check that *GeoServer satisfies SecurityService
	// and SecurityServiceWithContext.
	var _ SecurityService = (*GeoServer)(nil)
	var _ SecurityServiceWithContext = (*GeoServer)(nil)
	var _ ACLService = (*GeoServer)(nil)
	var _ ACLServiceWithContext = (*GeoServer)(nil)
}
