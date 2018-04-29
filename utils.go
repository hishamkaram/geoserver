package geoserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

//DoRequest :asdasd
func (g *GeoServer) DoRequest(request HTTPRequest) (responseText []byte, statusCode int) {
	var req *http.Request
	switch request.Method {
	case getMethod, deleteMethod:
		req := g.GetGeoserverRequest(request.URL, request.Method, request.Accept, nil, "")
	case postMethod, putMethod:
		req := g.GetGeoserverRequest(request.URL, request.Method, request.Accept, request.Data, request.DataType)
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
	response, responseErr := g.httpClient.Do(req)
	if responseErr != nil {
		panic(responseErr)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	g.logger.Infof("url:%s\nresponse Status=%s\n", req.URL, response.Status)
	return body, response.StatusCode
}

//GetError this return the proper error message
func (g *GeoServer) GetError(statusCode int, text []byte) (err error) {
	geoserverErr, ok := statusErrorMapping[statusCode]
	if !ok {
		geoserverErr = errors.New("Unexpected Error")
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
		g.logger.Fatal(err)
		return nil, err
	}
	return serializedStruct, nil
}

//DeSerializeJSON json struct to struct
func (g *GeoServer) DeSerializeJSON(response []byte, structObj interface{}) (err error) {
	err = json.Unmarshal(response, &structObj)
	if err != nil {
		g.logger.Fatal(err)
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
func (g *GeoServer) ParseURL(urlParts ...string) string {
	geoserverURL, err := url.Parse(g.ServerURL)
	if err != nil {
		log.Fatal(err)
	}
	urlArr := append([]string{geoserverURL.Path}, urlParts...)
	geoserverURL.Path = path.Join(urlArr...)
	return geoserverURL.String()

}
