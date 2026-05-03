package settings_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/settings"
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

// ---- Empty-Contact and Empty-JAIExt wire quirks ----

func TestContact_UnmarshalEmptyString(t *testing.T) {
	// GeoServer returns "contact":"" when no contact is configured.
	var s settings.ServiceSettings
	if err := json.Unmarshal([]byte(`{"contact":"","charset":"UTF-8"}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.Charset != "UTF-8" {
		t.Fatalf("Charset = %q", s.Charset)
	}
	// Contact pointer should still decode to a zero Contact (custom Unmarshal
	// keeps the zero value rather than failing).
	if s.Contact == nil {
		t.Fatalf("Contact is nil; expected zero Contact")
	}
	if s.Contact.ContactEmail != "" || s.Contact.ContactOrganization != "" {
		t.Fatalf("expected zero Contact, got %+v", s.Contact)
	}
}

func TestContact_UnmarshalObject(t *testing.T) {
	var s settings.ServiceSettings
	if err := json.Unmarshal([]byte(`{"contact":{"contactEmail":"admin@example.com","contactOrganization":"Acme"}}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.Contact == nil || s.Contact.ContactEmail != "admin@example.com" || s.Contact.ContactOrganization != "Acme" {
		t.Fatalf("Contact = %+v", s.Contact)
	}
}

func TestJAIExt_UnmarshalEmptyString(t *testing.T) {
	var j settings.JAI
	if err := json.Unmarshal([]byte(`{"allowInterpolation":true,"jaiext":""}`), &j); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !j.AllowInterpolation {
		t.Fatalf("AllowInterpolation = false")
	}
	if j.JAIExt == nil {
		t.Fatalf("JAIExt is nil; expected zero JAIExt")
	}
	if j.JAIExt.JAIExtOperations != nil {
		t.Fatalf("expected nil JAIExtOperations, got %+v", j.JAIExt.JAIExtOperations)
	}
}

// ---- Get / Update HTTP path tests ----

func TestGet_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/settings" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"global":{
			"settings":{"id":"global","charset":"UTF-8","numDecimals":8,"contact":""},
			"jai":{"allowInterpolation":true,"jaiext":""},
			"coverageAccess":{"maxPoolSize":10,"corePoolSize":5,"queueType":"UNBOUNDED"},
			"updateSequence":42,"globalServices":true
		}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	s, err := c.Settings.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Global.Settings == nil || s.Global.Settings.Charset != "UTF-8" || s.Global.Settings.NumDecimals != 8 {
		t.Fatalf("Settings = %+v", s.Global.Settings)
	}
	if s.Global.JAI == nil || !s.Global.JAI.AllowInterpolation {
		t.Fatalf("JAI = %+v", s.Global.JAI)
	}
	if s.Global.CoverageAccess == nil || s.Global.CoverageAccess.MaxPoolSize != 10 || s.Global.CoverageAccess.QueueType != "UNBOUNDED" {
		t.Fatalf("CoverageAccess = %+v", s.Global.CoverageAccess)
	}
	if s.Global.UpdateSequence != 42 || !s.Global.GlobalServices {
		t.Fatalf("Global = %+v", s.Global)
	}
}

func TestGet_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Settings.Get(context.Background())
	if !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrServerError, got %v", err)
	}
}

func TestUpdate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/settings" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"global":`) || !strings.Contains(s, `"charset":"UTF-16"`) {
			t.Errorf("body = %s", s)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Settings.Update(context.Background(), &settings.Settings{
		Global: settings.Global{
			Settings: &settings.ServiceSettings{Charset: "UTF-16"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NilSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Settings.Update(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "nil settings") {
		t.Fatalf("expected nil-settings error, got %v", err)
	}
}

func TestUpdate_400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Settings.Update(context.Background(), &settings.Settings{})
	if !errors.Is(err, geoserver.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}
