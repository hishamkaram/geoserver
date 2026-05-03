package transport

import (
	"fmt"
	"io"
	"net/http"
)

// Logger is the subset of *geoserver.Logger that the transport layer
// depends on. Defined here as a small interface so internal/transport
// doesn't import the root package.
type Logger interface {
	Errorf(format string, args ...any)
	Infof(format string, args ...any)
}

// Execute runs an already-built and -authenticated *http.Request against
// the supplied client, applies the optional query map, reads the response
// body, and returns body + status. On any failure (transport error,
// body-read error) it returns (nil, 0) or (nil, status) and logs the
// error via logger; callers should treat statusCode == 0 as a
// transport-level failure.
//
// A defer-recover translates any unexpected panic in the algorithm into
// the historical (string-of-panic, 0) contract callers may depend on.
//
// The req must already have a context attached (req.WithContext(ctx)),
// auth headers set (typically by SetBasicAuth in a request-builder), and
// any per-method body / Content-Type set. Execute does not mutate any
// of those.
func Execute(req *http.Request, client *http.Client, logger Logger, query map[string]string) (body []byte, statusCode int) {
	defer func() {
		if r := recover(); r != nil {
			body = fmt.Appendf(nil, "%s", r)
			statusCode = 0
		}
	}()

	if len(query) != 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	response, responseErr := client.Do(req)
	if responseErr != nil {
		logger.Errorf("DoRequest: %s %s: %v", req.Method, req.URL, responseErr)
		return nil, 0
	}
	defer func() { _ = response.Body.Close() }()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		logger.Errorf("DoRequest: read body for %s %s: %v", req.Method, req.URL, readErr)
		return nil, response.StatusCode
	}
	logger.Infof("url:%s  Status=%s", req.URL, response.Status)
	return body, response.StatusCode
}
