package geoserver

import (
	"bytes"
	"context"
	"fmt"
)

// SettingsService defines the GeoServer global-settings operations.
//
// Note: in v1.0.x the interface declared an `UpdateGlobalSettings(...)` method
// (plural) but the implementation was named `UpdateGlobalSetting` (singular),
// so any consumer of this interface failed to compile. v1.1 ships both names —
// callers should use the plural; the singular is kept for backward compatibility
// and marked Deprecated.
type SettingsService interface {
	// GetGlobalSettings returns the GeoServer global settings, or an error.
	GetGlobalSettings() (settings GlobalSettings, err error)

	// UpdateGlobalSettings performs a partial update of the GeoServer global
	// settings. Returns whether the change was applied and an error if any.
	UpdateGlobalSettings(globalSettings GlobalSettings) (modified bool, err error)
}

// SettingsServiceWithContext is the context-aware sibling of [SettingsService].
type SettingsServiceWithContext interface {
	GetGlobalSettingsContext(ctx context.Context) (settings GlobalSettings, err error)
	UpdateGlobalSettingsContext(ctx context.Context, globalSettings GlobalSettings) (modified bool, err error)
}

// Contact describes the GeoServer contact-information block surfaced under
// /rest/settings/contact.
type Contact struct {
	AddressCity         string `json:"addressCity,omitempty"`
	AddressCountry      string `json:"addressCountry,omitempty"`
	AddressType         string `json:"addressType,omitempty"`
	ContactEmail        string `json:"contactEmail,omitempty"`
	ContactOrganization string `json:"contactOrganization,omitempty"`
	ContactPerson       string `json:"contactPerson,omitempty"`
	ContactPosition     string `json:"contactPosition,omitempty"`
}

// Settings is the GeoServer service-settings block (charset, decimals,
// online-resource hints, etc.).
type Settings struct {
	Id                                 string      `json:"id,omitempty"`
	Contact                            interface{} `json:"contact,omitempty"`
	Charset                            string      `json:"charset,omitempty"`
	NumDecimals                        int         `json:"numDecimals,omitempty"`
	OnlineResource                     string      `json:"onlineResource,omitempty"`
	Verbose                            bool        `json:"verbose,omitempty"`
	VerboseExceptions                  bool        `json:"verboseExceptions,omitempty"`
	LocalWorkspaceIncludesPrefix       bool        `json:"localWorkspaceIncludesPrefix,omitempty"`
	ShowCreatedTimeColumnsInAdminList  bool        `json:"showCreatedTimeColumnsInAdminList,omitempty"`
	ShowModifiedTimeColumnsInAdminList bool        `json:"showModifiedTimeColumnsInAdminList,omitempty"`
}

// JaiExtOperations enumerates the JAI-Ext operations a GeoServer is configured
// to expose; surfaced under settings.jai.jaiext.jaiExtOperations.
type JaiExtOperations struct {
	Class  string   `json:"@class,omitempty"`
	String []string `json:"string,omitempty"`
}

// Jaiext wraps the JAI-Ext operation list reported by GeoServer.
type Jaiext struct {
	JaiExtOperations JaiExtOperations `json:"jaiExtOperations,omitempty"`
}

// Jai mirrors the JAI (Java Advanced Imaging) tunables reported under
// settings.jai. See https://docs.geoserver.org/stable/en/user/server/jai.html
type Jai struct {
	AllowInterpolation bool        `json:"allowInterpolation,omitempty"`
	Recycling          bool        `json:"recycling,omitempty"`
	TilePriority       int         `json:"tilePriority,omitempty"`
	MemoryCapacity     float32     `json:"memoryCapacity,omitempty"`
	MemoryThreshold    float32     `json:"memoryThreshold,omitempty"`
	ImageIOCache       bool        `json:"imageIOCache,omitempty"`
	PngAcceleration    bool        `json:"pngAcceleration,omitempty"`
	JpegAcceleration   bool        `json:"jpegAcceleration,omitempty"`
	AllowNativeMosaic  bool        `json:"allowNativeMosaic,omitempty"`
	AllowNativeWarp    bool        `json:"allowNativeWarp,omitempty"`
	Jaiext             interface{} `json:"jaiext,omitempty"`
}

// CoverageAccess controls the thread pool that processes raster (coverage)
// access; surfaced under settings.coverageAccess.
type CoverageAccess struct {
	MaxPoolSize           int    `json:"maxPoolSize,omitempty"`
	CorePoolSize          int    `json:"corePoolSize,omitempty"`
	KeepAliveTime         int    `json:"keepAliveTime,omitempty"`
	QueueType             string `json:"queueType,omitempty"`
	ImageIOCacheThreshold int    `json:"imageIOCacheThreshold,omitempty"`
}

// Global is the top-level "global" settings document returned by
// GET /rest/settings.json. It composes [Settings], [Jai], and [CoverageAccess].
type Global struct {
	Settings                    Settings       `json:"settings,omitempty"`
	Jai                         Jai            `json:"jai,omitempty"`
	CoverageAccess              CoverageAccess `json:"coverageAccess,omitempty"`
	UpdateSequence              int            `json:"updateSequence,omitempty"`
	FeatureTypeCacheSize        int            `json:"featureTypeCacheSize,omitempty"`
	GlobalServices              bool           `json:"globalServices,omitempty"`
	XmlPostRequestLogBufferSize int            `json:"xmlPostRequestLogBufferSize,omitempty"`
}

// GlobalSettings geoserver settings
type GlobalSettings struct {
	Global Global `json:"global,omitempty"`
}

// GetGlobalSettings returns global settings using context.Background.
func (g *GeoServer) GetGlobalSettings() (globalSettings GlobalSettings, err error) {
	return g.GetGlobalSettingsContext(context.Background())
}

// GetGlobalSettingsContext is the context-aware variant of [GeoServer.GetGlobalSettings].
func (g *GeoServer) GetGlobalSettingsContext(ctx context.Context) (globalSettings GlobalSettings, err error) {
	targetURL := g.ParseURL("rest", "settings")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		globalSettings = GlobalSettings{}
		err = g.GetError(responseCode, response)
		return
	}
	var settingsResponse GlobalSettings
	err = g.DeSerializeJSON(response, &settingsResponse)
	if err != nil {
		return GlobalSettings{}, err
	}
	globalSettings = settingsResponse
	return
}

// UpdateGlobalSettings performs a partial update of GeoServer global settings
// using context.Background.
func (g *GeoServer) UpdateGlobalSettings(globalSettings GlobalSettings) (modified bool, err error) {
	return g.UpdateGlobalSettingsContext(context.Background(), globalSettings)
}

// UpdateGlobalSettingsContext is the context-aware variant of [GeoServer.UpdateGlobalSettings].
func (g *GeoServer) UpdateGlobalSettingsContext(ctx context.Context, globalSettings GlobalSettings) (modified bool, err error) {
	targetURL := g.ParseURL("rest", "settings")
	serializedSettings, serErr := g.SerializeStruct(globalSettings)
	if serErr != nil {
		return false, fmt.Errorf("UpdateGlobalSettings: serialize settings: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedSettings),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		modified = false
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}

// UpdateGlobalSetting performs a partial update of GeoServer global settings.
//
// Deprecated: this name is retained for backward compatibility with v1.0.x;
// new code should call [GeoServer.UpdateGlobalSettings] (plural), which
// matches the [SettingsService] interface declaration.
func (g *GeoServer) UpdateGlobalSetting(globalSettings GlobalSettings) (modified bool, err error) {
	return g.UpdateGlobalSettings(globalSettings)
}
