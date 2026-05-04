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
)

func newPasswordTestClient(t *testing.T, srv *httptest.Server) *geoserver.Client {
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

func TestMasterPassword_Get_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/masterpw" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"oldMasterPassword":"super-secret"}`)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	got, err := c.Security.MasterPassword.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "super-secret" {
		t.Errorf("got %q, want %q", got, "super-secret")
	}
}

func TestMasterPassword_Update_BodyShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/security/masterpw" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		for _, want := range []string{`"oldMasterPassword":"old-pw"`, `"newMasterPassword":"new-pw"`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	if err := c.Security.MasterPassword.Update(context.Background(), "old-pw", "new-pw"); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestMasterPassword_Update_RejectsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	if err := c.Security.MasterPassword.Update(context.Background(), "", "new"); err == nil {
		t.Error("empty old: expected error")
	}
	if err := c.Security.MasterPassword.Update(context.Background(), "old", ""); err == nil {
		t.Error("empty new: expected error")
	}
}

func TestMasterPassword_Get_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	_, err := c.Security.MasterPassword.Get(context.Background())
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestSelfPassword_Change_BodyShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/security/self/password" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"newPassword":"new-pw"`) {
			t.Errorf("body missing newPassword field; got %s", s)
		}
		// The body MUST NOT carry an oldPassword — auth header proves possession.
		if strings.Contains(s, "oldPassword") {
			t.Errorf("body should not include oldPassword; got %s", s)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	if err := c.Security.SelfPassword.Change(context.Background(), "new-pw"); err != nil {
		t.Fatalf("Change: %v", err)
	}
}

func TestSelfPassword_Change_RejectsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	if err := c.Security.SelfPassword.Change(context.Background(), ""); err == nil {
		t.Error("expected empty-newPassword error")
	}
}

func TestSelfPassword_Change_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newPasswordTestClient(t, srv)
	err := c.Security.SelfPassword.Change(context.Background(), "new")
	if !errors.Is(err, geoserver.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
