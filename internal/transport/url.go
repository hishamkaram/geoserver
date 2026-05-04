// Package transport contains the HTTP transport algorithms for the v2
// GeoServer client. External code must not import this package.
package transport

import (
	"errors"
	"net/url"
	"path"
	"strings"
)

// ErrInvalidBaseURL is returned by [BuildURL] when the supplied base URL
// fails to parse.
var ErrInvalidBaseURL = errors.New("transport: invalid base URL")

// BuildURL joins parts onto base with each user-provided segment
// PathEscape'd individually. Empty segments are dropped.
//
// The encoded path is preserved through [url.URL.String] by setting
// [url.URL.RawPath] alongside [url.URL.Path]. Without RawPath, a segment
// PathEscape'd to a sequence containing "%" (e.g., "*" → "%2A") would be
// re-encoded by String() to "%252A", which GeoServer's StrictHttpFirewall
// rejects as a potentially malicious URL.
func BuildURL(base string, parts []string) (string, error) {
	u, err := url.Parse(base)
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
