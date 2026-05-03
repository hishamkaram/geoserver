package transport_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hishamkaram/geoserver/internal/transport"
)

// silentLogger drops every message; satisfies transport.Logger.
type silentLogger struct{}

func (silentLogger) Errorf(string, ...any) {}
func (silentLogger) Infof(string, ...any)  {}

func TestBuildURL_PlainSegments(t *testing.T) {
	got, err := transport.BuildURL("http://localhost:8080/geoserver/", []string{"rest", "workspaces", "topp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "http://localhost:8080/geoserver/rest/workspaces/topp"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildURL_DropsEmptySegments(t *testing.T) {
	got, err := transport.BuildURL("http://localhost:8080/geoserver/", []string{"rest", "workspaces", "", "topp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, "/rest/workspaces/topp") {
		t.Fatalf("expected suffix /rest/workspaces/topp, got %q", got)
	}
}

func TestBuildURL_NoDoubleEncoding(t *testing.T) {
	cases := []struct {
		name        string
		segs        []string
		mustContain string
		mustNot     string
	}{
		{
			name:        "asterisk",
			segs:        []string{"rest", "security", "acl", "layers", "ws.*.r"},
			mustContain: "ws.%2A.r",
			mustNot:     "%252A",
		},
		{
			name:        "space",
			segs:        []string{"rest", "workspaces", "my workspace"},
			mustContain: "my%20workspace",
			mustNot:     "%2520",
		},
		{
			name:        "non-ASCII",
			segs:        []string{"rest", "workspaces", "café"},
			mustContain: "%C3%A9",
			mustNot:     "%25",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := transport.BuildURL("http://localhost:8080/geoserver/", tc.segs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(got, tc.mustContain) {
				t.Fatalf("URL %q must contain %q", got, tc.mustContain)
			}
			if strings.Contains(got, tc.mustNot) {
				t.Fatalf("URL %q must NOT contain %q (regression: double-encoded)", got, tc.mustNot)
			}
		})
	}
}

func TestBuildURL_InvalidBaseURL(t *testing.T) {
	_, err := transport.BuildURL("://nope", []string{"rest"})
	if !errors.Is(err, transport.ErrInvalidBaseURL) {
		t.Fatalf("expected ErrInvalidBaseURL, got %v", err)
	}
}

func TestExecute_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("foo"); got != "bar" {
			t.Errorf("expected ?foo=bar, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/rest/workspaces", http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	body, status := transport.Execute(req, srv.Client(), silentLogger{}, map[string]string{"foo": "bar"})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestExecute_TransportError(t *testing.T) {
	// Build a request to a never-listening port; expect (nil, 0).
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1/dead", http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	client := &http.Client{} // no timeout — relies on the OS-level connection refusal
	body, status := transport.Execute(req, client, silentLogger{}, nil)
	if status != 0 {
		t.Fatalf("expected status 0 on transport error, got %d", status)
	}
	if body != nil {
		t.Fatalf("expected nil body on transport error, got %q", body)
	}
}
