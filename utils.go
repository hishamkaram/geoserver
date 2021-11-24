package geoserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
)

//HTTPRequest is an http request object
type HTTPRequest struct {
	URL      string
	Accept   string
	Query    map[string]string
	Data     io.Reader
	DataType string
	Method   string
}

//UtilsInterface contians common function used to help you deal with data and geoserver api
type UtilsInterface interface {
	DoRequest(request HTTPRequest) (responseText []byte, statusCode int)
	SerializeStruct(structObj interface{}) ([]byte, error)
	DeSerializeJSON(response []byte, structObj interface{}) (err error)
	ParseURL(urlParts ...string) (parsedURL string)
}

//DoRequest Send request and return result and statusCode
func (g *GeoServer) DoRequest(request HTTPRequest) (responseText []byte, statusCode int) {
	defer func() {
		if r := recover(); r != nil {
			responseText = []byte(fmt.Sprintf("%s", r))
			statusCode = 0
		}
	}()
	var req *http.Request
	switch request.Method {
	case getMethod, deleteMethod:
		req = g.GetGeoserverRequest(request.URL, request.Method, request.Accept, nil, "")
	case postMethod, putMethod:
		req = g.GetGeoserverRequest(request.URL, request.Method, request.Accept, request.Data, request.DataType)
	default:
		panic("unrecognized http request Method")
	}
	if len(request.Query) != 0 {
		q := req.URL.Query()
		for k, v := range request.Query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	response, responseErr := g.HttpClient.Do(req)
	if responseErr != nil {
		panic(responseErr)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	g.logger.Infof("url:%s  Status=%s", req.URL, response.Status)
	return body, response.StatusCode
}

//GetError this return the proper error message
func (g *GeoServer) GetError(statusCode int, text []byte) (err error) {
	geoserverErr, ok := statusErrorMapping[statusCode]
	if !ok {
		geoserverErr = fmt.Errorf("Unexpected Error with status code %d", statusCode)
	}
	errDetails := string(text)
	fullMSG := fmt.Sprintf("abstract:%s\ndetails:%s\n", geoserverErr, errDetails)
	return errors.New(fullMSG)
}

// IsEmpty helper function to check if obj/struct is nil/empty
func IsEmpty(object interface{}) bool {
	if object == nil {
		return true
	} else if object == "" {
		return true
	} else if object == false {
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

//SerializeStruct convert struct to json
func (g *GeoServer) SerializeStruct(structObj interface{}) ([]byte, error) {
	serializedStruct, err := json.Marshal(&structObj)
	if err != nil {
		g.logger.Error(err)
		return nil, err
	}
	return serializedStruct, nil
}

//DeSerializeJSON json struct to struct
func (g *GeoServer) DeSerializeJSON(response []byte, structObj interface{}) (err error) {
	err = json.Unmarshal(response, &structObj)
	if err != nil {
		g.logger.Error(err)
		return err
	}
	return nil
}
func (g *GeoServer) getGoGeoserverPackageDir() string {
	dir, err := filepath.Abs("./")
	if err != nil {
		panic(err)
	}
	return dir
}

//ParseURL this function join urlParts with geoserver url
func (g *GeoServer) ParseURL(urlParts ...string) (parsedURL string) {
	defer func() {
		if r := recover(); r != nil {
			parsedURL = ""
		}
	}()
	geoserverURL, err := url.Parse(g.ServerURL)
	if err != nil {
		g.logger.Error(err)
		panic(err)
	}
	urlArr := append([]string{geoserverURL.Path}, urlParts...)
	geoserverURL.Path = path.Join(urlArr...)
	parsedURL = geoserverURL.String()
	return

}

//requestResource does request, gets resource data and fill the response struct with parsed json
func (g *GeoServer) requestResource(targetURL string, response interface{}) (err error) {
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	responseData, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(responseData))
		err = g.GetError(responseCode, responseData)
		return
	}

	if err = json.Unmarshal(responseData, response); err != nil {
		return fmt.Errorf("can't parse respose from %v: %v", targetURL, err)
	}

	return
}

//createEntity does POST request to create a resource or entity
//checkError is a callback function processing the error, if nil the default error processing will perform
func (g *GeoServer) createEntity(targetURL string, entity interface{}, checkError func(statusCode int, response []byte) error) (created bool, err error) {

	var serializedLayer []byte
	if entity != nil {
		serializedLayer, _ = g.SerializeStruct(entity)
	} else {
		serializedLayer = []byte{}
	}

	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedLayer),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)

	if checkError == nil {
		if responseCode != statusCreated {
			g.logger.Error(string(response))
			err = g.GetError(responseCode, response)
			return
		}
	} else {
		err = checkError(responseCode, response)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

//deleteEntity does DELETE request to delete the entity
func (g *GeoServer) deleteEntity(targetURL string) (deleted bool, err error) {

	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		err = g.GetError(responseCode, response)
		return
	}

	return true, nil
}
