package geoserver

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for common GeoServer REST failure modes. Use with
// [errors.Is]:
//
//	if errors.Is(err, geoserver.ErrNotFound) { ... }
var (
	// ErrNotFound matches any 404 response from GeoServer.
	ErrNotFound = errors.New("geoserver: not found")
	// ErrUnauthorized matches any 401 response.
	ErrUnauthorized = errors.New("geoserver: unauthorized")
	// ErrForbidden matches any 403 response.
	ErrForbidden = errors.New("geoserver: forbidden")
	// ErrConflict matches any 409 response.
	ErrConflict = errors.New("geoserver: conflict")
	// ErrBadRequest matches any 400 response.
	ErrBadRequest = errors.New("geoserver: bad request")
	// ErrMethodNotAllowed matches any 405 response.
	ErrMethodNotAllowed = errors.New("geoserver: method not allowed")
	// ErrUnsupportedMediaType matches any 415 response.
	ErrUnsupportedMediaType = errors.New("geoserver: unsupported media type")
	// ErrRateLimited matches any 429 response.
	ErrRateLimited = errors.New("geoserver: rate limited")
	// ErrServerError matches any 5xx response.
	ErrServerError = errors.New("geoserver: server error")
)

// maxBodyBytes caps how much of a non-OK HTTP response body is preserved on
// an [*Error]. GeoServer error pages can be quite large (HTML stack traces);
// truncating prevents log/error-string explosions.
const maxBodyBytes = 8 * 1024

// Error is the typed error returned by *GeoServer when a REST call returns a
// non-success HTTP status. Match against the sentinel package vars
// ([ErrNotFound], [ErrServerError], etc.) via [errors.Is]:
//
//	if errors.Is(err, geoserver.ErrNotFound) {
//	    // workspace/layer/style/etc. doesn't exist
//	}
//
// Or unwrap to inspect the full transport details:
//
//	var apiErr *geoserver.Error
//	if errors.As(err, &apiErr) {
//	    log.Printf("status=%d url=%s body=%s", apiErr.StatusCode, apiErr.URL, apiErr.Body)
//	}
//
// The Error.Error() string format ("abstract:%s\ndetails:%s\n") is preserved
// from v1.0 so any code that previously matched on error message text
// continues to work unchanged.
type Error struct {
	// Op is the high-level operation that failed (e.g. "GetWorkspace").
	// May be empty if the construction site did not provide one.
	Op string
	// URL is the URL that produced the error.
	URL string
	// StatusCode is the HTTP status code returned by GeoServer (or 0 for
	// transport-level failures such as DNS or connection-refused).
	StatusCode int
	// Body is the response body, truncated to maxBodyBytes.
	Body []byte
	// Err is an optional wrapped underlying error.
	Err error
}

// Error returns the v1.0-compatible "abstract:%s\ndetails:%s\n" format.
func (e *Error) Error() string {
	geoserverErr, ok := statusErrorMapping[e.StatusCode]
	var label string
	if ok {
		label = geoserverErr.Error()
	} else {
		label = fmt.Sprintf("unexpected error with status code %d", e.StatusCode)
	}
	return fmt.Sprintf("abstract:%s\ndetails:%s\n", label, string(e.Body))
}

// Unwrap returns the wrapped underlying error, if any.
func (e *Error) Unwrap() error { return e.Err }

// Is reports whether the receiver matches one of the package sentinel errors.
// Status codes map to sentinels as follows:
//
//	400 -> ErrBadRequest
//	401 -> ErrUnauthorized
//	403 -> ErrForbidden
//	404 -> ErrNotFound
//	405 -> ErrMethodNotAllowed
//	409 -> ErrConflict
//	415 -> ErrUnsupportedMediaType
//	429 -> ErrRateLimited
//	5xx -> ErrServerError
func (e *Error) Is(target error) bool {
	switch target {
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case ErrForbidden:
		return e.StatusCode == http.StatusForbidden
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	case ErrMethodNotAllowed:
		return e.StatusCode == http.StatusMethodNotAllowed
	case ErrConflict:
		return e.StatusCode == http.StatusConflict
	case ErrUnsupportedMediaType:
		return e.StatusCode == http.StatusUnsupportedMediaType
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrServerError:
		return e.StatusCode >= 500 && e.StatusCode < 600
	}
	return false
}

// newError builds an [*Error] for a non-success HTTP response, truncating
// the body to maxBodyBytes.
func newError(op, url string, statusCode int, body []byte) *Error {
	if len(body) > maxBodyBytes {
		body = body[:maxBodyBytes]
	}
	return &Error{Op: op, URL: url, StatusCode: statusCode, Body: body}
}
