package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
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

// Request is the per-call shape: method, fully-built URL, optional
// body (JSON or raw), optional query params, optional Accept override.
//
// The transport does NOT own URL composition — callers (resource
// sub-clients) build the URL via [BuildURL] and pass it here.
//
// Body and RawBody are mutually exclusive: when RawBody is non-nil
// it is sent as-is with ContentType (defaulting to
// "application/octet-stream") and Body is ignored.
type Request struct {
	Method string
	URL    string
	// Body, if non-nil, is JSON-encoded.
	// Use nil for GET / DELETE / HEAD with no body, or set RawBody
	// for non-JSON payloads.
	Body any
	// RawBody, if non-nil, is sent as-is with ContentType. When set,
	// Body is ignored and the request is NOT JSON-encoded.
	RawBody io.Reader
	// ContentType is the wire Content-Type for RawBody. Ignored when
	// RawBody is nil. Defaults to "application/octet-stream" when
	// RawBody is set and ContentType is empty.
	ContentType string
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
	return doRequest(ctx, doer, logger, op, req, out)
}

// DoXML sends a request and decodes the response body as XML. Use
// this for the OWS endpoints (WMS / WFS / WCS GetCapabilities) where
// the response is XML rather than JSON. Request body, if any, is sent
// as RawBody — typical OWS calls are GET with no body.
//
// On success: out (if non-nil) is xml.Unmarshal'd from the response
// body; the body cap is [xmlBodyReadCap] (32 MiB) since capabilities
// documents are often well above the [bodyReadCap] used by [DoJSON].
//
// On non-2xx: returns a structured Error wrapping the response body
// (truncated to [bodyReadCap]) so the caller can inspect what came back.
//
// On transport error: status is 0; err is the wrapped transport error.
func DoXML(ctx context.Context, doer Doer, logger *slog.Logger, op string, req Request, out any) (status int, err error) {
	if req.Accept == "" {
		req.Accept = "application/xml"
	}
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, bodyReadCap))
		logDebug(logger, "request done", "op", op, "method", req.Method, "url", req.URL, "status", resp.StatusCode, "body_bytes", len(body))
		return resp.StatusCode, &Error{
			Op:         op,
			Method:     req.Method,
			URL:        req.URL,
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, xmlBodyReadCap))
	if readErr != nil {
		logDebug(logger, "body read failed", "op", op, "method", req.Method, "url", req.URL, "status", resp.StatusCode, "err", readErr)
		return resp.StatusCode, fmt.Errorf("%s: read body: %w", op, readErr)
	}
	logDebug(logger, "request done", "op", op, "method", req.Method, "url", req.URL, "status", resp.StatusCode, "body_bytes", len(body))

	if out == nil || len(body) == 0 {
		return resp.StatusCode, nil
	}
	if xmlErr := xml.Unmarshal(body, out); xmlErr != nil {
		return resp.StatusCode, fmt.Errorf("%s: decode response: %w", op, xmlErr)
	}
	return resp.StatusCode, nil
}

// DoRaw sends a request with an arbitrary-Reader body and explicit
// Content-Type / Accept. The response handling is identical to
// [DoJSON] — out (if non-nil) is JSON-decoded; non-2xx returns a
// *Error.
//
// Use this for non-JSON uploads (SLD XML, shapefile zips, GeoTIFF
// blobs) where the request body is bytes and the response is the
// usual JSON / empty body.
//
// If body is nil, the request is sent with no payload. If contentType
// is empty, "application/octet-stream" is used. If accept is empty,
// "application/json" is used.
func DoRaw(ctx context.Context, doer Doer, logger *slog.Logger, op string, method, url string, body io.Reader, contentType, accept string, query map[string]string, out JSON) (status int, err error) {
	return doRequest(ctx, doer, logger, op, Request{
		Method:      method,
		URL:         url,
		RawBody:     body,
		ContentType: contentType,
		Accept:      accept,
		Query:       query,
	}, out)
}

func doRequest(ctx context.Context, doer Doer, logger *slog.Logger, op string, req Request, out JSON) (status int, err error) {
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

// xmlBodyReadCap is the larger cap used by [DoXML] on the success path.
// WMS / WFS / WCS GetCapabilities documents are commonly tens to
// hundreds of KiB on real installations; the JSON 8 KiB cap is too
// small. The error path still uses [bodyReadCap] so an oversized error
// body doesn't blow up.
const xmlBodyReadCap = 32 << 20 // 32 MiB

// buildHTTPRequest constructs the http.Request with body / query /
// Accept set. Auth and User-Agent are applied by the transport
// RoundTripper, not here.
func buildHTTPRequest(ctx context.Context, req Request) (*http.Request, error) {
	var (
		body        io.Reader = http.NoBody
		contentType string
	)
	switch {
	case req.RawBody != nil:
		body = req.RawBody
		contentType = req.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	case req.Body != nil:
		buf, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}
		body = bytes.NewReader(buf)
		contentType = "application/json"
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
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
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
