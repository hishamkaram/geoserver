package geoserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	"github.com/hishamkaram/geoserver/internal/transport"
)

// HTTPRequest is an http request object
type HTTPRequest struct {
	URL      string
	Accept   string
	Query    map[string]string
	Data     io.Reader
	DataType string
	Method   string
}

// UtilsInterface contains common function used to help you deal with data and geoserver api
type UtilsInterface interface {
	DoRequest(request HTTPRequest) (responseText []byte, statusCode int)
	DoRequestContext(ctx context.Context, request HTTPRequest) (responseText []byte, statusCode int)
	SerializeStruct(structObj interface{}) ([]byte, error)
	DeSerializeJSON(response []byte, structObj interface{}) (err error)
	ParseURL(urlParts ...string) (parsedURL string)
}

// DoRequest sends an HTTP request to GeoServer and returns the response body
// and HTTP status code. On any failure (transport error, unsupported method,
// nil request) it returns (nil, 0); errors are surfaced via the logger and
// callers should treat statusCode == 0 as a transport-level failure.
//
// Backward-compatibility note: prior versions of this method panicked on
// transport errors. Panics are now translated to (nil, 0) returns.
//
// DoRequest uses context.Background. Use [GeoServer.DoRequestContext] when
// you need cancellation, deadlines, or trace propagation.
func (g *GeoServer) DoRequest(request HTTPRequest) (responseText []byte, statusCode int) {
	return g.DoRequestContext(context.Background(), request)
}

// DoRequestContext is the context-aware variant of [GeoServer.DoRequest]. The
// supplied context is attached to the underlying *http.Request and honoured
// by transport, including cancellation and deadlines.
//
// Implementation: the per-request shape (method, URL, body, auth) is built
// here on *GeoServer because it depends on the client's credentials. The
// generic "apply query, execute, read body, log" portion is delegated to
// internal/transport.Execute so it can be unit-tested in isolation and
// reused by v2.
func (g *GeoServer) DoRequestContext(ctx context.Context, request HTTPRequest) (responseText []byte, statusCode int) {
	var (
		req    *http.Request
		reqErr error
	)
	switch request.Method {
	case getMethod, deleteMethod:
		req, reqErr = g.GetGeoserverRequestE(request.URL, request.Method, request.Accept, nil, "")
	case postMethod, putMethod:
		req, reqErr = g.GetGeoserverRequestE(request.URL, request.Method, request.Accept, request.Data, request.DataType)
	default:
		g.logger.Errorf("DoRequest: unsupported HTTP method %q", request.Method)
		return nil, 0
	}
	if reqErr != nil {
		g.logger.Errorf("DoRequest: build request %s %s: %v", request.Method, request.URL, reqErr)
		return nil, 0
	}
	if req == nil {
		g.logger.Errorf("DoRequest: failed to construct request for %s %s", request.Method, request.URL)
		return nil, 0
	}
	req = req.WithContext(ctx)

	return transport.Execute(req, g.HttpClient, g.logger, request.Query)
}

// GetError returns a typed [*Error] for the given GeoServer HTTP response.
//
// The returned error's Error() string preserves the v1.0
// "abstract:%s\ndetails:%s\n" format for backward compatibility, while new
// callers can match it against package sentinel errors:
//
//	err := gs.CreateWorkspace("topp")
//	if errors.Is(err, geoserver.ErrNotFound) { ... }
//	var apiErr *geoserver.Error
//	if errors.As(err, &apiErr) { ... apiErr.StatusCode ... apiErr.Body ... }
func (g *GeoServer) GetError(statusCode int, text []byte) (err error) {
	return newError("", "", statusCode, text)
}

// IsEmpty helper function to check if obj/struct is nil/empty
func IsEmpty(object interface{}) bool {
	switch object {
	case nil:
		return true
	case "":
		return true
	case false:
		return true
	}
	if reflect.ValueOf(object).Kind() == reflect.Struct {
		empty := reflect.New(reflect.TypeOf(object)).Elem().Interface()
		if reflect.DeepEqual(object, empty) {
			return true
		}
	}
	return false
}

// SerializeStruct convert struct to json
func (g *GeoServer) SerializeStruct(structObj interface{}) ([]byte, error) {
	serializedStruct, err := json.Marshal(&structObj)
	if err != nil {
		g.logger.Error(err)
		return nil, err
	}
	return serializedStruct, nil
}

// DeSerializeJSON json struct to struct
func (g *GeoServer) DeSerializeJSON(response []byte, structObj interface{}) (err error) {
	err = json.Unmarshal(response, &structObj)
	if err != nil {
		g.logger.Error(err)
		return err
	}
	return nil
}

// ParseURL joins urlParts with the GeoServer base URL, applying url.PathEscape
// to each user-provided segment. Empty segments are dropped. On a malformed
// base URL, it returns the empty string and logs at Error.
//
// Behavior change in v1.1.0: each path segment is now PathEscape'd, so
// workspace/layer names containing spaces, slashes, or non-ASCII characters
// produce correct URLs instead of malformed ones. Previously such inputs
// silently produced bad URLs.
//
// Bug fix in v1.1.x: the encoded path is preserved through url.URL.String()
// by setting [url.URL.RawPath] alongside [url.URL.Path]. Without RawPath,
// segments that PathEscape'd to a sequence containing "%" (e.g., "*" → "%2A")
// were re-encoded by String() to "%252A", which GeoServer's request firewall
// rejects as a potentially malicious URL.
//
// The algorithm itself lives in internal/transport.BuildURL; this method is
// a thin wrapper that translates errors into the logged-and-empty-string
// contract v1.0 callers expect.
func (g *GeoServer) ParseURL(urlParts ...string) (parsedURL string) {
	defer func() {
		if r := recover(); r != nil {
			parsedURL = ""
		}
	}()

	parsed, err := transport.BuildURL(g.ServerURL, urlParts)
	if err != nil {
		if errors.Is(err, transport.ErrInvalidBaseURL) {
			g.logger.Errorf("ParseURL: invalid base URL %q", g.ServerURL)
		} else {
			g.logger.Errorf("ParseURL: cannot build path: %v", err)
		}
		return ""
	}
	return parsed
}
