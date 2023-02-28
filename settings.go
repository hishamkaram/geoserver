package geoserver

import "bytes"

// SettingsService define geoserver settings operations
type SettingsService interface {

	// GetGlobalSettings get global settings else return error
	GetGlobalSettings() (settings *GlobalSettings, err error)

	// UpdateGlobalSettings partial update global geoserver settings else return error
	UpdateGlobalSettings(globalSettings GlobalSettings) (modified bool, err error)
}

type Contact struct {
	AddressCity         string `json:"addressCity,omitempty"`
	AddressCountry      string `json:"addressCountry,omitempty"`
	AddressType         string `json:"addressType,omitempty"`
	ContactEmail        string `json:"contactEmail,omitempty"`
	ContactOrganization string `json:"contactOrganization,omitempty"`
	ContactPerson       string `json:"contactPerson,omitempty"`
	ContactPosition     string `json:"contactPosition,omitempty"`
}

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

type JaiExtOperations struct {
	Class  string   `json:"@class,omitempty"`
	String []string `json:"string,omitempty"`
}

type Jaiext struct {
	JaiExtOperations JaiExtOperations `json:"jaiExtOperations,omitempty"`
}

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

type CoverageAccess struct {
	MaxPoolSize           int    `json:"maxPoolSize,omitempty"`
	CorePoolSize          int    `json:"corePoolSize,omitempty"`
	KeepAliveTime         int    `json:"keepAliveTime,omitempty"`
	QueueType             string `json:"queueType,omitempty"`
	ImageIOCacheThreshold int    `json:"imageIOCacheThreshold,omitempty"`
}

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

// GetGlobalSettings get global settings else return error
func (g *GeoServer) GetGlobalSettings() (globalSettings GlobalSettings, err error) {
	targetURL := g.ParseURL("rest", "settings")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
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

// UpdateGlobalSetting partial update global geoserver settings else return error
func (g *GeoServer) UpdateGlobalSetting(globalSettings GlobalSettings) (modified bool, err error) {
	targetURL := g.ParseURL("rest", "settings")
	serializedSettings, _ := g.SerializeStruct(globalSettings)
	httpRequest := HTTPRequest{
		Method:   putMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(serializedSettings),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequest(httpRequest)
	if responseCode != statusOk {
		g.logger.Error(string(response))
		modified = false
		err = g.GetError(responseCode, response)
		return
	}
	modified = true
	return
}
