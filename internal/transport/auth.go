package transport

import (
	"net/http"
)

// AuthRoundTripper wraps another [http.RoundTripper] and attaches an
// auth header to every outgoing request.
//
// The Apply func is called once per request. It receives the request
// (with Header already lazily-allocated if needed) and sets whatever
// header(s) the auth scheme requires. Designed to be wrapped further by
// users who want to layer additional behavior (OpenTelemetry spans,
// Vault-rotated creds, retry libs).
type AuthRoundTripper struct {
	Apply func(*http.Request)
	Base  http.RoundTripper
}

// RoundTrip implements [http.RoundTripper].
//
// To stay friendly to higher-layer wrappers (e.g., [http.Client] retries
// or future request rewinds), the request is cloned before mutation —
// the caller's copy stays untouched.
func (rt *AuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.Apply == nil {
		base := rt.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	rt.Apply(clone)
	base := rt.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

// HeaderRoundTripper wraps another [http.RoundTripper] and attaches a
// fixed set of headers to every outgoing request. Used for User-Agent
// and any headers configured via WithHeader.
type HeaderRoundTripper struct {
	Headers http.Header
	Base    http.RoundTripper
}

// RoundTrip implements [http.RoundTripper].
func (rt *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(rt.Headers) == 0 {
		base := rt.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	for key, values := range rt.Headers {
		// Don't clobber an explicit per-request override.
		if clone.Header.Get(key) != "" {
			continue
		}
		for _, v := range values {
			clone.Header.Add(key, v)
		}
	}
	base := rt.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}
