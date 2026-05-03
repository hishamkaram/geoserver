package security_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/security"
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

// ---- Service scope accessor ----

func TestService_DefaultsToDefault(t *testing.T) {
	c, err := geoserver.New("http://localhost:8080/geoserver")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.Security.Users().Service(); got != security.DefaultService {
		t.Fatalf("Users().Service() = %q, want %q", got, security.DefaultService)
	}
	if got := c.Security.UsersInService("").Service(); got != security.DefaultService {
		t.Fatalf("UsersInService(\"\").Service() = %q, want %q", got, security.DefaultService)
	}
	if got := c.Security.UsersInService("custom").Service(); got != "custom" {
		t.Fatalf("UsersInService(\"custom\").Service() = %q", got)
	}
	if got := c.Security.GroupsInService("custom").Service(); got != "custom" {
		t.Fatalf("Groups().Service() = %q", got)
	}
}

// ---- Users ----

func TestUsers_List_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/usergroup/service/default/users" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"users":[{"userName":"alice","enabled":true},{"userName":"bob","enabled":false}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	users, err := c.Security.Users().List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 || users[0].Name != "alice" || users[0].Enabled != true || users[1].Enabled != false {
		t.Fatalf("Users = %+v", users)
	}
}

func TestUsers_List_CustomService(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/usergroup/service/jdbc/users" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"users":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Security.UsersInService("jdbc").List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsers_Create_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/usergroup/service/default/users" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, sub := range []string{
			`"user":`, `"userName":"alice"`, `"enabled":true`, `"password":"secret"`,
		} {
			if !strings.Contains(s, sub) {
				t.Errorf("body missing %q\nbody: %s", sub, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Users().Create(context.Background(), &security.User{
		Name: "alice", Enabled: true, Password: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsers_Create_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Users().Create(context.Background(), &security.User{
		Name: "dup", Password: "x",
	})
	if !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestUsers_Create_NilUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Users().Create(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil user") {
		t.Fatalf("expected nil-user error, got %v", err)
	}
}

func TestUsers_Create_EmptyPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Users().Create(context.Background(), &security.User{Name: "alice"})
	if err == nil || !strings.Contains(err.Error(), "empty user Password") {
		t.Fatalf("expected empty-password error, got %v", err)
	}
}

func TestUsers_Delete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/security/usergroup/service/default/user/alice" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.Users().Delete(context.Background(), "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Groups (cross-version response shape) ----

func TestGroups_List_NewShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 2.28+ shape.
		_, _ = io.WriteString(w, `{"groups":["admins","editors"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Security.Groups().List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "admins" || got[1].Name != "editors" {
		t.Fatalf("Groups = %+v", got)
	}
}

func TestGroups_List_OldShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// older 2.x shape.
		_, _ = io.WriteString(w, `{"groupNames":["legacy_group"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Security.Groups().List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "legacy_group" {
		t.Fatalf("Groups = %+v", got)
	}
}

func TestGroups_Create_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/usergroup/service/default/group/admins" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.Groups().Create(context.Background(), "admins"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGroups_Delete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/security/usergroup/service/default/group/admins" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.Groups().Delete(context.Background(), "admins"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Roles (cross-version response shape) ----

func TestRoles_List_NewShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/roles" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"roles":["ROLE_ADMIN","ROLE_EDITOR"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Security.Roles.List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "ROLE_ADMIN" {
		t.Fatalf("Roles = %v", got)
	}
}

func TestRoles_List_OldShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"roleNames":["LEGACY_ROLE"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Security.Roles.List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "LEGACY_ROLE" {
		t.Fatalf("Roles = %v", got)
	}
}

func TestRoles_Create_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/roles/role/ROLE_NEW" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.Roles.Create(context.Background(), "ROLE_NEW"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoles_Delete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/rest/security/roles/role/ROLE_OLD" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.Roles.Delete(context.Background(), "ROLE_OLD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Roles.ForUser ----

func TestRoles_ForUser_NewShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/roles/user/alice" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"roles":["ROLE_ADMIN"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Security.Roles.ForUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "ROLE_ADMIN" {
		t.Fatalf("ForUser = %v", got)
	}
}

func TestRoles_ForUser_OldShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"roleNames":["LEGACY"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Security.Roles.ForUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "LEGACY" {
		t.Fatalf("ForUser = %v", got)
	}
}

func TestRoles_ForUser_EmptyUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Security.Roles.ForUser(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "empty userName") {
		t.Fatalf("expected empty-user error, got %v", err)
	}
}

// ---- Roles.AssignToUser / UnassignFromUser ----

func TestRoles_AssignToUser_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/rest/security/roles/role/ROLE_ADMIN/user/alice" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		// GeoServer returns 200 OK for assignment (not 201).
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Roles.AssignToUser(context.Background(), "ROLE_ADMIN", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoles_UnassignFromUser_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete ||
			r.URL.Path != "/rest/security/roles/role/ROLE_ADMIN/user/alice" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Roles.UnassignFromUser(context.Background(), "ROLE_ADMIN", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoles_AssignToUser_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.Roles.AssignToUser(context.Background(), "MISSING", "alice")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ---- URL escaping ----

func TestUsers_List_URLEscaping_Service(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"users":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Security.UsersInService("svc*1").List(context.Background(), security.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURI, "svc%2A1") {
		t.Fatalf("expected single-encoded segment, got %q", capturedURI)
	}
	if strings.Contains(capturedURI, "%252A") {
		t.Fatalf("URL is double-encoded: %q", capturedURI)
	}
}
