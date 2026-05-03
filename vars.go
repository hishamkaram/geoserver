package geoserver

import (
	"errors"
	"net/http"
)

const (
	statusOk                   = http.StatusOK                   // 200
	statusCreated              = http.StatusCreated              // 201
	statusBadRequest           = http.StatusBadRequest           // 400
	statusUnauthorized         = http.StatusUnauthorized         // 401
	statusForbidden            = http.StatusForbidden            // 403
	statusNotFound             = http.StatusNotFound             // 404
	statusNotAllowed           = http.StatusMethodNotAllowed     // 405
	statusConflict             = http.StatusConflict             // 409
	statusUnsupportedMediaType = http.StatusUnsupportedMediaType // 415
	statusTooManyRequests      = http.StatusTooManyRequests      // 429
	statusInternalError        = http.StatusInternalServerError  // 500
	statusBadGateway           = http.StatusBadGateway           // 502
	statusServiceUnavailable   = http.StatusServiceUnavailable   // 503
	statusGatewayTimeout       = http.StatusGatewayTimeout       // 504

	jsonType          = "application/json"
	zipType           = "application/zip"
	appXMLType        = "application/xml"
	xmlType           = "text/xml"
	sldType           = "application/vnd.ogc.sld+xml"
	contentTypeHeader = "Content-Type"
	acceptHeader      = "Accept"
	getMethod         = http.MethodGet
	putMethod         = http.MethodPut
	postMethod        = http.MethodPost
	deleteMethod      = http.MethodDelete
)

// statusErrorMapping maps HTTP status codes to short error labels surfaced via
// (*GeoServer).GetError. The full body is appended on top of these labels.
var statusErrorMapping = map[int]error{
	statusBadRequest:           errors.New("Bad Request"),
	statusUnauthorized:         errors.New("Unauthorized"),
	statusForbidden:            errors.New("Forbidden"),
	statusNotFound:             errors.New("Not Found"),
	statusNotAllowed:           errors.New("Method Not Allowed"),
	statusConflict:             errors.New("Conflict"),
	statusUnsupportedMediaType: errors.New("Unsupported Media Type"),
	statusTooManyRequests:      errors.New("Too Many Requests"),
	statusInternalError:        errors.New("Internal Server Error"),
	statusBadGateway:           errors.New("Bad Gateway"),
	statusServiceUnavailable:   errors.New("Service Unavailable"),
	statusGatewayTimeout:       errors.New("Gateway Timeout"),
}
