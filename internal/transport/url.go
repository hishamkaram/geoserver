// Package transport contains the HTTP transport algorithms used by the
// public GeoServer client. External code must not import this package.
//
// The functions here are thin enough that they don't take a *GeoServer —
// the public methods on *GeoServer are wrappers that pass in the relevant
// fields.
package transport

import (
	"errors"
	"net/url"
	"path"
	"strings"
)

// ErrInvalidBaseURL is returned by [BuildURL] when the supplied serverURL
// does not parse. The public ParseURL wrapper translates this into a
// logged error and returns an empty string for v1.0 source-compatibility.
var ErrInvalidBaseURL = errors.New("transport: invalid base URL")

// BuildURL joins parts onto serverURL with each user-provided segment
// PathEscape'd individually. Empty segments are dropped.
//
// The encoded path is preserved through url.URL.String() by setting
// [url.URL.RawPath] alongside [url.URL.Path]; without RawPath, a segment
// PathEscape'd to a sequence containing "%" (e.g., "*" → "%2A") would be
// re-encoded by String() to "%252A", which GeoServer's StrictHttpFirewall
// rejects as a potentially malicious URL. See utils_unit_test.go
// TestParseURL_NoDoubleEncoding for the regression guard.
//
// Returns ErrInvalidBaseURL if serverURL fails to parse. Returns the
// original Go url.PathUnescape error if the built path can't be
// unescaped (should be unreachable since the path is built from
// PathEscape outputs, but treated as a logged failure rather than a
// panic).
func BuildURL(serverURL string, parts []string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", ErrInvalidBaseURL
	}

	basePath := strings.TrimRight(u.Path, "/")
	escaped := make([]string, 0, len(parts)+1)
	if basePath != "" {
		escaped = append(escaped, basePath)
	}
	for _, p := range parts {
		if p == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(p))
	}
	rawPath := path.Join(escaped...)
	decoded, decodeErr := url.PathUnescape(rawPath)
	if decodeErr != nil {
		return "", decodeErr
	}
	u.Path = decoded
	u.RawPath = rawPath
	return u.String(), nil
}
