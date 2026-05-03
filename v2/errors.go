package geoserver

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors. APIError.Is wraps these so callers can match status
// codes via errors.Is(err, ErrNotFound) etc.
//
// The set mirrors v1's sentinel list; the Error type itself is renamed
// (v1: *Error → v2: *APIError) and tightened — v2 does not preserve v1's
// "abstract:%s\ndetails:%s\n" format because v2 is a clean break.
var (
	ErrBadRequest           = errors.New("geoserver: bad request")
	ErrUnauthorized         = errors.New("geoserver: unauthorized")
	ErrForbidden            = errors.New("geoserver: forbidden")
	ErrNotFound             = errors.New("geoserver: not found")
	ErrMethodNotAllowed     = errors.New("geoserver: method not allowed")
	ErrConflict             = errors.New("geoserver: conflict")
	ErrUnsupportedMediaType = errors.New("geoserver: unsupported media type")
	ErrRateLimited          = errors.New("geoserver: rate limited")
	ErrServerError          = errors.New("geoserver: internal server error")
	ErrBadGateway           = errors.New("geoserver: bad gateway")
	ErrServiceUnavailable   = errors.New("geoserver: service unavailable")
	ErrGatewayTimeout       = errors.New("geoserver: gateway timeout")
)

// statusToSentinel maps HTTP status codes to package sentinel errors.
var statusToSentinel = map[int]error{
	http.StatusBadRequest:           ErrBadRequest,
	http.StatusUnauthorized:         ErrUnauthorized,
	http.StatusForbidden:            ErrForbidden,
	http.StatusNotFound:             ErrNotFound,
	http.StatusMethodNotAllowed:     ErrMethodNotAllowed,
	http.StatusConflict:             ErrConflict,
	http.StatusUnsupportedMediaType: ErrUnsupportedMediaType,
	http.StatusTooManyRequests:      ErrRateLimited,
	http.StatusInternalServerError:  ErrServerError,
	http.StatusBadGateway:           ErrBadGateway,
	http.StatusServiceUnavailable:   ErrServiceUnavailable,
	http.StatusGatewayTimeout:       ErrGatewayTimeout,
}

// APIError represents a non-2xx response from GeoServer. The single error
// type for the v2 module — every wire-level failure surfaces as
// *APIError so callers can match either by sentinel ([errors.Is]) or by
// inspecting fields ([errors.As]).
type APIError struct {
	// Op identifies the public operation that produced the error
	// (e.g., "Workspaces.Create"). Set by the resource-client
	// wrapper. Useful for logging.
	Op string

	// URL is the request URL.
	URL string

	// Method is the HTTP method.
	Method string

	// StatusCode is the HTTP status code from GeoServer.
	StatusCode int

	// Body is the response body, truncated to a fixed cap (8 KiB)
	// to avoid unbounded retention. Useful for diagnostics; do not
	// parse for control flow — use errors.Is against the package
	// sentinels instead.
	Body []byte
}

// Error returns a stable, parseable message of the form
//
//	geoserver: <Op> <Method> <URL>: <statusCode> <statusText>: <body-preview>
//
// Body preview is truncated to ~120 bytes.
func (e *APIError) Error() string {
	preview := string(e.Body)
	if len(preview) > 120 {
		preview = preview[:120] + "…"
	}
	op := e.Op
	if op == "" {
		op = "request"
	}
	return fmt.Sprintf("geoserver: %s %s %s: %d %s: %s",
		op, e.Method, e.URL, e.StatusCode, http.StatusText(e.StatusCode), preview)
}

// Unwrap returns the sentinel for this status code, enabling errors.Is.
func (e *APIError) Unwrap() error {
	if sentinel, ok := statusToSentinel[e.StatusCode]; ok {
		return sentinel
	}
	return nil
}

// HTTPStatusCode returns the HTTP status code that produced the
// error. A stable accessor (in addition to the [APIError.StatusCode]
// field) so sub-clients in v2/rest/* can branch on the status without
// importing the root package — that would create an import cycle
// since the root imports each rest/<resource>.
func (e *APIError) HTTPStatusCode() int { return e.StatusCode }

// Is reports whether target matches the sentinel for this APIError's
// status code. Lets callers use errors.Is(err, ErrNotFound) directly on
// an *APIError.
func (e *APIError) Is(target error) bool {
	if sentinel, ok := statusToSentinel[e.StatusCode]; ok && sentinel == target {
		return true
	}
	return false
}

// newAPIError constructs an *APIError, truncating the body to bodyCap.
const bodyCap = 8 << 10 // 8 KiB

func newAPIError(op, method, url string, statusCode int, body []byte) *APIError {
	if len(body) > bodyCap {
		body = body[:bodyCap]
	}
	return &APIError{
		Op:         op,
		URL:        url,
		Method:     method,
		StatusCode: statusCode,
		Body:       body,
	}
}
