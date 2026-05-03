package geoserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"strings"
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
func (g *GeoServer) DoRequestContext(ctx context.Context, request HTTPRequest) (responseText []byte, statusCode int) {
	defer func() {
		// Belt-and-suspenders: in case any code path below still panics,
		// translate to the historical (string-of-panic, 0) contract that
		// callers may depend on.
		if r := recover(); r != nil {
			responseText = []byte(fmt.Sprintf("%s", r))
			statusCode = 0
		}
	}()

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

	if len(request.Query) != 0 {
		q := req.URL.Query()
		for k, v := range request.Query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	response, responseErr := g.HttpClient.Do(req)
	if responseErr != nil {
		g.logger.Errorf("DoRequest: %s %s: %v", req.Method, req.URL, responseErr)
		return nil, 0
	}
	defer func() { _ = response.Body.Close() }()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		g.logger.Errorf("DoRequest: read body for %s %s: %v", req.Method, req.URL, readErr)
		return nil, response.StatusCode
	}
	g.logger.Infof("url:%s  Status=%s", req.URL, response.Status)
	return body, response.StatusCode
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

// getGoGeoserverPackageDir returns the absolute path to the package working
// directory. Used by tests; library callers should not rely on it.
func (g *GeoServer) getGoGeoserverPackageDir() (string, error) {
	return filepath.Abs("./")
}

// ParseURL joins urlParts with the GeoServer base URL, applying url.PathEscape
// to each user-provided segment. Empty segments are dropped. On a malformed
// base URL, it returns the empty string and logs at Error.
//
// Behavior change in v1.1.0: each path segment is now PathEscape'd, so
// workspace/layer names containing spaces, slashes, or non-ASCII characters
// produce correct URLs instead of malformed ones. Previously such inputs
// silently produced bad URLs.
func (g *GeoServer) ParseURL(urlParts ...string) (parsedURL string) {
	defer func() {
		if r := recover(); r != nil {
			parsedURL = ""
		}
	}()

	geoserverURL, err := url.Parse(g.ServerURL)
	if err != nil {
		g.logger.Errorf("ParseURL: invalid base URL %q: %v", g.ServerURL, err)
		return ""
	}

	// Preserve the base path (e.g. "/geoserver/"), then escape each
	// caller-provided segment individually. Empty segments are skipped so
	// callers can pass conditional values without producing "//".
	basePath := strings.TrimRight(geoserverURL.Path, "/")
	escaped := make([]string, 0, len(urlParts)+1)
	if basePath != "" {
		escaped = append(escaped, basePath)
	}
	for _, p := range urlParts {
		if p == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(p))
	}
	geoserverURL.Path = path.Join(escaped...)
	return geoserverURL.String()
}
