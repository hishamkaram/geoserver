package resources

import (
	"errors"
	"net/http"
)

// httpStatusErr is satisfied by the parent package's *APIError. We
// can't import the root v2 package directly (it imports this
// sub-package, so doing so would create a cycle), so we declare the
// interface locally and rely on errors.As to walk the wrapping
// chain.
type httpStatusErr interface {
	error
	HTTPStatusCode() int
}

// isNotFound reports whether err is a 404 from the wrapped HTTP
// request, regardless of which sub-client surfaced it.
func isNotFound(err error) bool {
	var s httpStatusErr
	return errors.As(err, &s) && s.HTTPStatusCode() == http.StatusNotFound
}
