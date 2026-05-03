package geoserver

import (
	"strings"
	"testing"
)

// TestParseURL_NoDoubleEncoding is a regression guard for the v1.1.x bug
// where url.URL.String() re-encoded already-PathEscape'd segments — e.g.
// "*" → "%2A" → "%252A" — which GeoServer's StrictHttpFirewall then
// rejected. The fix sets [url.URL.RawPath] alongside [url.URL.Path] so the
// encoding is preserved verbatim.
func TestParseURL_NoDoubleEncoding(t *testing.T) {
	gs := New("http://localhost:8080/geoserver/", "u", "p", WithLogger(nil))

	cases := []struct {
		name        string
		segments    []string
		mustContain string
		mustNot     string
	}{
		{
			name:        "asterisk in segment",
			segments:    []string{"rest", "security", "acl", "layers", "ws.*.r"},
			mustContain: "ws.%2A.r",
			mustNot:     "%252A",
		},
		{
			name:        "space in workspace name",
			segments:    []string{"rest", "workspaces", "my workspace"},
			mustContain: "my%20workspace",
			mustNot:     "%2520",
		},
		{
			name:        "non-ASCII",
			segments:    []string{"rest", "workspaces", "café"},
			mustContain: "%C3%A9",
			mustNot:     "%25",
		},
		{
			name:        "plain ASCII unaffected",
			segments:    []string{"rest", "workspaces", "topp"},
			mustContain: "/rest/workspaces/topp",
			mustNot:     "%",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := gs.ParseURL(tc.segments...)
			if !strings.Contains(got, tc.mustContain) {
				t.Fatalf("URL %q must contain %q", got, tc.mustContain)
			}
			if strings.Contains(got, tc.mustNot) {
				t.Fatalf("URL %q must NOT contain %q (regression: double-encoded)", got, tc.mustNot)
			}
		})
	}
}

func TestParseURL_DropsEmptySegments(t *testing.T) {
	gs := New("http://localhost:8080/geoserver/", "u", "p", WithLogger(nil))
	got := gs.ParseURL("rest", "workspaces", "", "topp")
	if got != "http://localhost:8080/geoserver/rest/workspaces/topp" {
		t.Fatalf("unexpected URL: %q", got)
	}
}
