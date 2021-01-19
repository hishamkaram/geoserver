package geoserver

import (
	"errors"
	"net/http"
)

const (
	jsonType          = "application/json"
	zipType           = "application/zip"
	appXMLType        = "application/xml"
	xmlType           = "text/xml"
	sldType           = "application/vnd.ogc.sld+xml"
	contentTypeHeader = "Content-Type"
	acceptHeader      = "Accept"
)

var statusErrorMapping = map[int]error{
	http.StatusMethodNotAllowed:    errors.New(http.StatusText(http.StatusMethodNotAllowed)),
	http.StatusNotFound:            errors.New(http.StatusText(http.StatusNotFound)),
	http.StatusUnauthorized:        errors.New(http.StatusText(http.StatusUnauthorized)),
	http.StatusInternalServerError: errors.New(http.StatusText(http.StatusInternalServerError)),
	http.StatusForbidden:           errors.New(http.StatusText(http.StatusForbidden)),
}
