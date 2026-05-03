package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Doer is the subset of [*http.Client] the transport layer needs.
// Allows tests to substitute a fake.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Request is the per-call shape: method, fully-built URL, optional JSON
// body, optional query params, optional Accept override.
//
// The transport does NOT own URL composition — callers (resource
// sub-clients) build the URL via [BuildURL] and pass it here.
type Request struct {
	Method string
	URL    string
	// Body, if non-nil, is JSON-encoded with the body cap respected.
	// Use nil for GET / DELETE / HEAD with no body.
	Body any
	// Query is added on top of any query already present in URL.
	Query map[string]string
	// Accept overrides the default "application/json".
	Accept string
}

// JSON is a JSON-shaped target for [DoJSON]. nil disables decoding.
type JSON any

// DoJSON sends a JSON-shaped request, decodes a JSON-shaped response,
// and translates non-2xx responses into a *transport-layer-Error.
//
// On success: status code is reported in the returned status; out (if
// non-nil) is filled from the response body. Error is nil.
//
// On non-2xx: returns a structured Error wrapping the response body
// (capped) so the calling resource sub-client can rewrap as a
// public *APIError with the correct Op string. status is the actual
// status code; out is unchanged.
//
// On transport error: status is 0; err is the wrapped transport error.
func DoJSON(ctx context.Context, doer Doer, logger *slog.Logger, op string, req Request, out JSON) (status int, err error) {
	httpReq, err := buildHTTPRequest(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("%s: build request: %w", op, err)
	}

	resp, err := doer.Do(httpReq)
	if err != nil {
		logDebug(logger, "request failed", "op", op, "method", req.Method, "url", req.URL, "err", err)
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, bodyReadCap))
	if readErr != nil {
		logDebug(logger, "body read failed", "op", op, "method", req.Method, "url", req.URL, "status", resp.StatusCode, "err", readErr)
		return resp.StatusCode, fmt.Errorf("%s: read body: %w", op, readErr)
	}

	logDebug(logger, "request done", "op", op, "method", req.Method, "url", req.URL, "status", resp.StatusCode, "body_bytes", len(body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, &Error{
			Op:         op,
			Method:     req.Method,
			URL:        req.URL,
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	if out == nil || len(body) == 0 {
		return resp.StatusCode, nil
	}
	if jsonErr := json.Unmarshal(body, out); jsonErr != nil {
		return resp.StatusCode, fmt.Errorf("%s: decode response: %w", op, jsonErr)
	}
	return resp.StatusCode, nil
}

// Error is the transport-layer non-2xx error. The public *APIError type
// in the root v2 package wraps this with the same fields and the
// status-to-sentinel mapping.
type Error struct {
	Op         string
	Method     string
	URL        string
	StatusCode int
	Body       []byte
}

func (e *Error) Error() string {
	preview := string(e.Body)
	if len(preview) > 120 {
		preview = preview[:120] + "…"
	}
	return fmt.Sprintf("%s %s %s: %d %s: %s",
		e.Op, e.Method, e.URL, e.StatusCode, http.StatusText(e.StatusCode), preview)
}

// Read body cap matches the public *APIError.Body cap so the wrapper
// doesn't have to re-truncate.
const bodyReadCap = 8 << 10 // 8 KiB

// buildHTTPRequest constructs the http.Request with body / query /
// Accept set. Auth and User-Agent are applied by the transport
// RoundTripper, not here.
func buildHTTPRequest(ctx context.Context, req Request) (*http.Request, error) {
	var body io.Reader = http.NoBody
	if req.Body != nil {
		buf, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return nil, err
	}

	accept := req.Accept
	if accept == "" {
		accept = "application/json"
	}
	httpReq.Header.Set("Accept", accept)
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if len(req.Query) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.Query {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	return httpReq, nil
}

func logDebug(logger *slog.Logger, msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Debug(msg, args...)
}
