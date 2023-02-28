package geoserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetGlobalSettings(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	settings, err := gsCatalog.GetGlobalSettings()
	assert.NotNil(t, settings)
	assert.Nil(t, err)
}

func TestUpdateGlobalSetting(t *testing.T) {
	// Get the initial settings of this geoserver
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	initialSettings, err := gsCatalog.GetGlobalSettings()
	assert.NotNil(t, initialSettings)
	assert.Nil(t, err)

	updateSettings := initialSettings
	updateSettings.Global.Settings.NumDecimals = 200

	// Update the settings
	gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	modified, err := gsCatalog.UpdateGlobalSetting(updateSettings)

	assert.True(t, modified)
	assert.Nil(t, err)

	// Check if the update succeeded
	gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	updatedSettings, err := gsCatalog.GetGlobalSettings()
	assert.NotNil(t, updatedSettings)
	assert.Nil(t, err)

	assert.Equal(t, 200, updatedSettings.Global.Settings.NumDecimals)

	// Reset settings to initial ones
	gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	updatedModified, err := gsCatalog.UpdateGlobalSetting(initialSettings)

	assert.True(t, updatedModified)
	assert.Nil(t, err)
}
