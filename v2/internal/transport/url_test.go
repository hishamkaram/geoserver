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
