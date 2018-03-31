package geoserver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
)

//DoGet helper function to create get request
func (g *GeoServer) DoGet(url string, accept string, query map[string]string) ([]byte, int) {
	req, err := g.GetGeoserverRequest(url, getMethod, accept, nil, "")
	if err != nil {
		panic(err)
	}
	if len(query) != 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	response, responseErr := g.HTTPClient.Do(req)
	if responseErr != nil {
		panic(responseErr)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != statusOk {
		fmt.Printf("%s \n", string(body))
	}
	fmt.Printf("%s \t response Status:%s \n", url, response.Status)
	return body, response.StatusCode
}

//DoPost helper function to create post request
func (g *GeoServer) DoPost(url string, data io.Reader, dataType string, accept string) ([]byte, int) {
	req, err := g.GetGeoserverRequest(url, postMethod, accept, data, dataType)
	if err != nil {
		panic(err)
	}
	response, responseErr := g.HTTPClient.Do(req)
	if responseErr != nil {
		panic(responseErr)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != statusCreated {
		fmt.Printf("%s \n", string(body))
	}
	fmt.Printf("%s \t response Status:%s \n", url, response.Status)
	return body, response.StatusCode
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

//DoPut helper function to create put request
func (g *GeoServer) DoPut(url string, data io.Reader, dataType string, accept string) ([]byte, int) {
	req, err := g.GetGeoserverRequest(url, putMethod, accept, data, dataType)
	if err != nil {
		panic(err)
	}
	response, responseErr := g.HTTPClient.Do(req)
	if responseErr != nil {
		panic(responseErr)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != statusOk {
		fmt.Printf("%s \n", string(body))
	}
	fmt.Printf("%s \t response Status:%s \n", url, response.Status)
	return body, response.StatusCode

}

//DoDelete helper function to create put request
func (g *GeoServer) DoDelete(url string, accept string, query map[string]string) ([]byte, int) {
	req, err := g.GetGeoserverRequest(url, deleteMethod, accept, nil, "")
	if err != nil {
		panic(err)
	}
	if len(query) != 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	response, responseErr := g.HTTPClient.Do(req)
	if responseErr != nil {
		panic(responseErr)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != statusOk {
		fmt.Printf("%s \n", string(body))
	}
	fmt.Printf("%s \t response Status:%s \n", url, response.Status)
	return body, response.StatusCode

}

//SerializeStruct convert struct to json
func (g *GeoServer) SerializeStruct(structObj interface{}) ([]byte, error) {
	serializedStruct, err := json.Marshal(&structObj)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return serializedStruct, nil
}

//DeSerializeJSON json struct to struct
func (g *GeoServer) DeSerializeJSON(response []byte, structObj interface{}) (err error) {
	err = json.Unmarshal(response, &structObj)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
