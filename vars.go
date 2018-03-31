package geoserver

import "errors"

const (
	statusOk           = 200
	statusCreated      = 201
	statusNotAllowed   = 405
	statusNotFound     = 404
	statusUnauthorized = 401
	jsonType           = "application/json"
	zipType            = "application/zip"
	appXMLType         = "application/xml"
	xmlType            = "text/xml"
	sldType            = "application/vnd.ogc.sld+xml"
	contentTypeHeader  = "Content-Type"
	acceptHeader       = "Accept"
	getMethod          = "GET"
	putMethod          = "PUT"
	postMethod         = "POST"
	deleteMethod       = "DELETE"
)

var statusErrorMapping = map[int]error{
	statusNotAllowed:   errors.New("Method Not Allowed"),
	statusNotFound:     errors.New("Not Found"),
	statusUnauthorized: errors.New("Unauthorized"),
}
