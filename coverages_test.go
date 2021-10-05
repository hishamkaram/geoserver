package geoserver

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var gsCatalog *GeoServer

var (
	coveragesTestWorkspace      = "sf"
	coveragesTestCoverageName   = "sfdem"
	coveragesTestStoreName      = "sfdem_test"
	coveragesTestDummyStoreName = "sfdem_dummy"

	coveragesTestStoreFile = map[string]string{
		coveragesTestStoreName:      "file:data/sf/sfdem.tif",
		coveragesTestDummyStoreName: "file:data/sf/dummy.tif",
	}
)

func before() {
	if gsCatalog == nil {
		gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	}
}

func coveragesPrepareTestStorage(t *testing.T, storeName string) {
	//creating coverageStore if doesn't exist

	coverageStore := CoverageStore{
		Name:        storeName,
		Description: storeName + " Description",
		Type:        "GeoTIFF",
		URL:         coveragesTestStoreFile[storeName],
		Workspace: &Resource{
			Name: coveragesTestWorkspace,
		},
		Enabled: true,
	}
	_, err := gsCatalog.CreateCoverageStore("sf", coverageStore)
	if err != nil && !strings.Contains(err.Error(), "exists") {
		assert.Fail(t, "can't create coverage store", err.Error())
	}

}

func coveragesRemoveCoverage(t *testing.T) {
	_, err := gsCatalog.DeleteCoverage(coveragesTestWorkspace, coveragesTestCoverageName, true)
	if err != nil {
		assert.Fail(t, "can't delete coverage", err.Error())
	}
}

func TestPublishCoverage(t *testing.T) {
	before()

	//preparing
	coveragesPrepareTestStorage(t, coveragesTestStoreName)

	_, err := gsCatalog.GetCoverage(coveragesTestWorkspace, coveragesTestCoverageName)
	if err == nil {
		coveragesRemoveCoverage(t)
	}

	done, err := gsCatalog.PublishCoverage(coveragesTestWorkspace, coveragesTestStoreName, coveragesTestCoverageName)
	assert.True(t, done)
	assert.Nil(t, err)
	done, errFail := gsCatalog.PublishCoverage(coveragesTestWorkspace, coveragesTestStoreName, "dummy")
	assert.False(t, done)
	assert.NotNil(t, errFail)

	coveragesRemoveCoverage(t)
}

/*
func TestPublishGeoTiffLayer(t *testing.T) {
	before()

	_, err := gsCatalog.PublishGeoTiffLayer(coveragesTestWorkspace, coveragesTestStoreName + "1", coveragesTestCoverageName, coveragesTestStoreFile[coveragesTestStoreName])

	if err != nil {
		assert.Fail(t, "can't publish geoTiff", err.Error())
	}

	return
}
*/

func TestGetCoverage(t *testing.T) {

	before()

	//preparing
	coveragesPrepareTestStorage(t, coveragesTestStoreName)

	_, err := gsCatalog.PublishCoverage(coveragesTestWorkspace, coveragesTestStoreName, coveragesTestCoverageName)
	if err != nil && !strings.Contains(err.Error(), "exists") {
		assert.Fail(t, "can't publish the coverage", err.Error())
	}

	//do the test
	coverage, err := gsCatalog.GetCoverage(coveragesTestWorkspace, coveragesTestCoverageName)
	assert.NotNil(t, coverage)
	assert.Nil(t, err)
	coverageFail, errFail := gsCatalog.GetCoverage(coveragesTestWorkspace, "dummy")
	assert.Nil(t, coverageFail)
	assert.NotNil(t, errFail)

	coveragesRemoveCoverage(t)
}

func TestGetCoverages(t *testing.T) {
	before()

	//preparing
	coveragesPrepareTestStorage(t, coveragesTestStoreName)

	_, err := gsCatalog.GetCoverage(coveragesTestWorkspace, coveragesTestCoverageName)
	if err == nil {
		coveragesRemoveCoverage(t)
	}

	coverages, err := gsCatalog.GetCoverages(coveragesTestWorkspace)
	if err != nil {
		assert.Fail(t, "can't get coverages list", err.Error())
	}
	assert.True(t, len(coverages) == 0)

	_, err = gsCatalog.PublishCoverage(coveragesTestWorkspace, coveragesTestStoreName, coveragesTestCoverageName)
	if err != nil {
		assert.Fail(t, "can't get publish the coverage", err.Error())
	}

	coverages, err = gsCatalog.GetCoverages(coveragesTestWorkspace)
	if err != nil {
		assert.Fail(t, "can't get coverages list", err.Error())
	}
	assert.True(t, len(coverages) == 1)

	coveragesRemoveCoverage(t)
}

func TestGetStorageCoverages(t *testing.T) {
	before()

	//preparing
	coveragesPrepareTestStorage(t, coveragesTestStoreName)

	coverages, err := gsCatalog.GetStoreCoverages(coveragesTestWorkspace, coveragesTestStoreName)
	if err != nil {
		assert.Fail(t, "can't get coverages list", err.Error())
	}
	assert.True(t, len(coverages) == 1)

	//create wrong storage
	coveragesPrepareTestStorage(t, coveragesTestDummyStoreName)

	//we have to get the error while reading wrong storage
	coverages, err = gsCatalog.GetStoreCoverages(coveragesTestWorkspace, coveragesTestDummyStoreName)
	assert.NotNil(t, err)

	_, err = gsCatalog.DeleteCoverageStore(coveragesTestWorkspace, coveragesTestDummyStoreName, true)
	if err != nil {
		assert.Fail(t, "can't delete temporary coverageStore", err.Error())
	}

}
