package geoserver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
)

//DoGet helper function to create get request
func (g *GeoServer) DoGet(url string, accept string) ([]byte, int) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	if accept != "" {
		req.Header.Add("Accept", fmt.Sprintf("%s", accept))
	}
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		panic(httpErr)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 201 {
			fmt.Printf("%s \n", string(body))
		}
		fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
		return body, resp.StatusCode
	}
}

//DoPost helper function to create post request
func (g *GeoServer) DoPost(url string, data []byte, dataType string) ([]byte, int) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	req.Header.Add("Content-Type", fmt.Sprintf("%s; charset=utf-8", dataType))
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		panic(httpErr)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 201 {
			fmt.Printf("%s \n", string(body))
		}
		fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
		return body, resp.StatusCode
	}

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
func (g *GeoServer) DoPut(url string, data []byte, dataType string) ([]byte, int) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(g.Username, g.Password)
	req.Header.Add("Content-Type", fmt.Sprintf("%s", dataType))
	req.Header.Set("Accept", "application/xml")
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		panic(httpErr)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 201 {
			fmt.Printf("%s \n", string(body))
		}
		fmt.Printf("%s \t response Status:%s \n", url, resp.Status)
		return body, resp.StatusCode
	}

}
