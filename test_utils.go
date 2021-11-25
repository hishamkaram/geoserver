package geoserver

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var gsCatalog *GeoServer

var testConfig *testEnv

type testEnv struct {
	Geoserver struct {
		Workspace string `yaml:"workspace"`
		ServerURL string `yaml:"geoserver_url"`
		Username  string `yaml:"username"`
		Password  string `yaml:"password"`
	} `yaml:"geoserver"`

	Postgres struct {
		Name   string
		Host   string
		Port   int
		Type   string
		DBName string
		DBUser string
		DBPass string
	} `yaml:"postgres"`

	PostgresJNDI struct {
		Name              string
		Type              string
		JndiReferenceName string
		Options           []Entry
	} `yaml:"postgresJNDI"`
}

func test_before(t *testing.T) {
	if err := test_load_env(); err != nil {
		t.Fatalf("can't load testing confiration from file: %v", err.Error())
	}
	if gsCatalog == nil {
		gsCatalog = GetCatalog(testConfig.Geoserver.ServerURL, testConfig.Geoserver.Username, testConfig.Geoserver.Password)
	}
}

func test_load_env() error {

	if testConfig != nil {
		return nil
	}

	testConfig = &testEnv{}

	yamlFile, err := ioutil.ReadFile(filepath.Join(gsCatalog.getGoGeoserverPackageDir(), "test_env.yml"))
	if err != nil {
		return fmt.Errorf("yamlFile.Get err %v ", err)
	}

	err = yaml.Unmarshal(yamlFile, testConfig)
	if err != nil {
		return fmt.Errorf("Unmarshal: %v", err)
	}
	return nil
}
