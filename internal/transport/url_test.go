package transport_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/transport"
)

func TestBuildURL_Plain(t *testing.T) {
	got, err := transport.BuildURL("http://localhost:8080/geoserver/", []string{"rest", "workspaces"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://localhost:8080/geoserver/rest/workspaces" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildURL_NoDoubleEncoding(t *testing.T) {
	cases := []struct{ name, seg, must, mustNot string }{
		{"asterisk", "*", "%2A", "%252A"},
		{"space", "my workspace", "my%20workspace", "%2520"},
		{"non-ASCII", "café", "%C3%A9", "%25"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := transport.BuildURL("http://localhost:8080/geoserver/", []string{"rest", tc.seg})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tc.must) {
				t.Fatalf("URL %q must contain %q", got, tc.must)
			}
			if strings.Contains(got, tc.mustNot) {
				t.Fatalf("URL %q must NOT contain %q (regression)", got, tc.mustNot)
			}
		})
	}
}

func TestBuildURL_DropsEmptySegments(t *testing.T) {
	got, err := transport.BuildURL("http://localhost:8080/geoserver/", []string{"rest", "", "workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://localhost:8080/geoserver/rest/workspaces" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildURL_InvalidBaseURL(t *testing.T) {
	_, err := transport.BuildURL("://nope", []string{"rest"})
	if !errors.Is(err, transport.ErrInvalidBaseURL) {
		t.Fatalf("expected ErrInvalidBaseURL, got %v", err)
	}
}

// FuzzBuildURL exercises the path-joining + escaping algorithm with
// arbitrary string segments. The property under test is the safety
// contract: BuildURL must not panic on any input — it should return an
// error or a well-formed URL string. v1 had two production bugs in this
// area (issue #22 and a separate URL-escaping regression); fuzzing here
// is genuinely defensive, not just a Scorecard checkbox.
func FuzzBuildURL(f *testing.F) {
	f.Add("http://localhost:8080/geoserver/", "rest", "workspaces", "topp")
	f.Add("https://geoserver.example.com/", "rest", "workspaces", "topp:states")
	f.Add("http://localhost/", "rest", "workspaces", "my workspace")
	f.Add("", "", "", "")
	f.Add("://malformed", "rest", "x", "y")

	f.Fuzz(func(t *testing.T, base, p1, p2, p3 string) {
		_, _ = transport.BuildURL(base, []string{p1, p2, p3})
	})
}
