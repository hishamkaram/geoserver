package security_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/security"
)

// ---- AuthProviders ----

func TestAuthProviders_List_ArrayShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/authproviders" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"authproviders":[
			{"id":"1","name":"default","className":"o.g.s.auth.UPAP","userGroupServiceName":"default"},
			{"id":"2","name":"corp","className":"o.g.s.auth.LDAPAP","userGroupServiceName":"default","serverURL":"ldaps://corp"}
		]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.Security.AuthProviders.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d", len(list))
	}
	if list[1].Name != "corp" || list[1].Extras["serverURL"] != "ldaps://corp" {
		t.Errorf("provider[1] = %+v / extras=%+v", list[1], list[1].Extras)
	}
}

func TestAuthProviders_List_ClassKeyedMapShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"authproviders":{"o.g.s.config.UsernamePasswordAuthenticationProviderConfig":{"id":"1","name":"default","className":"o.g.s.auth.UPAP","userGroupServiceName":"default"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.Security.AuthProviders.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "default" {
		t.Errorf("list = %+v", list)
	}
}

func TestAuthProviders_Roundtrip_FlatJSONWithExtras(t *testing.T) {
	// Verify Marshal/Unmarshal round-trip preserves provider-specific
	// extras alongside typed core fields.
	src := security.AuthProvider{
		Name: "corp", ClassName: "X", UserGroupServiceName: "default",
		Extras: map[string]interface{}{
			"serverURL":  "ldaps://corp",
			"userFormat": "uid={0}",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Body is the flat JSON; assert all four keys are emitted.
		s := string(body)
		for _, want := range []string{`"name":"corp"`, `"className":"X"`, `"userGroupServiceName":"default"`, `"serverURL":"ldaps://corp"`, `"userFormat":"uid={0}"`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.AuthProviders.Create(context.Background(), &src, security.CreateAuthProviderOptions{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestAuthProviders_Get_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Security.AuthProviders.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAuthProviders_SetOrder_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/security/authproviders/order" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"order":["a","b"]`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.AuthProviders.SetOrder(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatalf("SetOrder: %v", err)
	}
}

func TestAuthProviders_SetOrder_EmptyRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Security.AuthProviders.SetOrder(context.Background(), nil); err == nil {
		t.Fatal("expected empty-list error")
	}
}

// ---- AuthFilters ----

func TestAuthFilters_List_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"authfilters":{"authfilter":[
			{"name":"anonymous","href":"http://srv/rest/security/authfilters/anonymous.json"},
			{"name":"basic","href":"http://srv/rest/security/authfilters/basic.json"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.Security.AuthFilters.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].Name != "anonymous" {
		t.Errorf("list = %+v", list)
	}
}

func TestAuthFilters_Get_FlatExtras(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"my-oidc","className":"o.g.s.f.OIDCFilter","clientId":"id123","clientSecret":"sek"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	f, err := c.Security.AuthFilters.Get(context.Background(), "my-oidc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if f.Name != "my-oidc" || f.Extras["clientId"] != "id123" {
		t.Errorf("filter = %+v", f)
	}
}

func TestAuthFilters_Get_NullWireQuirkMapsToNotFound(t *testing.T) {
	// GeoServer wire-quirk: missing auth filter returns 200 with
	// {"null":""} body instead of 404. SDK translates to ErrNotFound.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"null":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Security.AuthFilters.Get(context.Background(), "missing")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for null wire shape, got %v", err)
	}
}

func TestAuthFilters_Get_ClassEnvelopeUnwrap(t *testing.T) {
	// GeoServer wraps single auth filter in class-name-keyed envelope.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"o.g.s.config.AnonymousAuthenticationFilterConfig":{"id":"x","name":"anonymous","className":"o.g.s.f.GeoServerAnonymousAuthenticationFilter"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	f, err := c.Security.AuthFilters.Get(context.Background(), "anonymous")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if f.Name != "anonymous" || f.ClassName != "o.g.s.f.GeoServerAnonymousAuthenticationFilter" {
		t.Errorf("filter = %+v", f)
	}
}

// ---- FilterChains ----

func TestFilterChains_List_AttributeStyleOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/filterchain.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"filterchain":{"filters":[
			{"@name":"web","@class":"o.g.s.HtmlLoginFilterChain","@path":"/web/**","@disabled":false,"@allowSessionCreation":true,"@ssl":false,"@matchHTTPMethod":false,"filter":["rememberme","form","anonymous"]},
			{"@name":"webLogin","@class":"o.g.s.ConstantFilterChain","@path":"/j_spring/","@disabled":false,"@allowSessionCreation":true,"@ssl":false,"@matchHTTPMethod":false,"filter":"form"}
		]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	list, err := c.Security.FilterChains.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d", len(list))
	}
	if list[0].Name != "web" || len(list[0].Filters) != 3 {
		t.Errorf("chain[0] = %+v", list[0])
	}
	// Single-string filter collapse.
	if list[1].Name != "webLogin" || len(list[1].Filters) != 1 || list[1].Filters[0] != "form" {
		t.Errorf("chain[1] single-filter collapse = %+v", list[1])
	}
}

func TestFilterChains_Get_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/filterchain/web.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"filters":{"@name":"web","@class":"o.g.s.HtmlLoginFilterChain","@path":"/web/**","filter":["form","anonymous"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	fc, err := c.Security.FilterChains.Get(context.Background(), "web")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.Name != "web" || fc.Path != "/web/**" || len(fc.Filters) != 2 {
		t.Errorf("chain = %+v", fc)
	}
}

func TestFilterChains_Create_BodyShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/security/filterchain" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		// Body must be the {"filters":{...}} envelope with @-attrs.
		for _, want := range []string{`"filters":{`, `"@name":"new"`, `"@path":"/x/**"`, `"@class":"o.g.s.X"`, `"filter":["a","b"]`} {
			if !strings.Contains(s, want) {
				t.Errorf("body missing %s; got %s", want, s)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.FilterChains.Create(context.Background(), &security.FilterChain{
		Name:      "new",
		ClassName: "o.g.s.X",
		Path:      "/x/**",
		Filters:   []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestFilterChains_SetOrder_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/security/filterchain/order" {
			t.Errorf("path = %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"order":["web","rest","default"]`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Security.FilterChains.SetOrder(context.Background(), []string{"web", "rest", "default"})
	if err != nil {
		t.Fatalf("SetOrder: %v", err)
	}
}
