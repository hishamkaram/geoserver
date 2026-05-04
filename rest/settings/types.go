// Package settings is the v2 sub-client for the GeoServer
// /rest/settings resource — the global server configuration document
// covering service metadata (charset, online-resource), JAI tunables,
// and coverage-access tuning.
package settings

import (
	"bytes"
	"encoding/json"
)

// Settings is the top-level settings document. The wire form wraps
// the [Global] block in a "global" key.
type Settings struct {
	Global Global `json:"global,omitempty"`
}

// Global composes the major settings groups returned by
// /rest/settings.json.
type Global struct {
	Settings                    *ServiceSettings `json:"settings,omitempty"`
	JAI                         *JAI             `json:"jai,omitempty"`
	CoverageAccess              *CoverageAccess  `json:"coverageAccess,omitempty"`
	UpdateSequence              int              `json:"updateSequence,omitempty"`
	FeatureTypeCacheSize        int              `json:"featureTypeCacheSize,omitempty"`
	GlobalServices              bool             `json:"globalServices,omitempty"`
	XMLPostRequestLogBufferSize int              `json:"xmlPostRequestLogBufferSize,omitempty"`
}

// ServiceSettings holds the service-level configuration block
// (charset, decimals, online-resource, contact info).
type ServiceSettings struct {
	ID                                 string   `json:"id,omitempty"`
	Contact                            *Contact `json:"contact,omitempty"`
	Charset                            string   `json:"charset,omitempty"`
	NumDecimals                        int      `json:"numDecimals,omitempty"`
	OnlineResource                     string   `json:"onlineResource,omitempty"`
	Verbose                            bool     `json:"verbose,omitempty"`
	VerboseExceptions                  bool     `json:"verboseExceptions,omitempty"`
	LocalWorkspaceIncludesPrefix       bool     `json:"localWorkspaceIncludesPrefix,omitempty"`
	ShowCreatedTimeColumnsInAdminList  bool     `json:"showCreatedTimeColumnsInAdminList,omitempty"`
	ShowModifiedTimeColumnsInAdminList bool     `json:"showModifiedTimeColumnsInAdminList,omitempty"`
}

// Contact is the contact-information block returned under
// settings.contact.
//
// Wire-format quirk: GeoServer returns `"contact":""` (a bare
// string) when no contact is configured, instead of omitting the
// field. The custom [Contact.UnmarshalJSON] tolerates the empty
// string and yields a zero-value Contact.
type Contact struct {
	AddressCity         string `json:"addressCity,omitempty"`
	AddressCountry      string `json:"addressCountry,omitempty"`
	AddressType         string `json:"addressType,omitempty"`
	ContactEmail        string `json:"contactEmail,omitempty"`
	ContactOrganization string `json:"contactOrganization,omitempty"`
	ContactPerson       string `json:"contactPerson,omitempty"`
	ContactPosition     string `json:"contactPosition,omitempty"`
}

// UnmarshalJSON tolerates GeoServer's empty-contact wire form. When
// the field is the bare string "" or null, the Contact decodes to its
// zero value rather than failing.
func (c *Contact) UnmarshalJSON(data []byte) error {
	d := bytes.TrimSpace(data)
	if len(d) == 0 || string(d) == "null" || string(d) == `""` {
		return nil
	}
	type alias Contact
	return json.Unmarshal(d, (*alias)(c))
}

// JAI tracks GeoServer's Java Advanced Imaging tuning knobs (under
// settings.jai). See https://docs.geoserver.org/stable/en/user/server/jai.html
type JAI struct {
	AllowInterpolation bool    `json:"allowInterpolation,omitempty"`
	Recycling          bool    `json:"recycling,omitempty"`
	TilePriority       int     `json:"tilePriority,omitempty"`
	MemoryCapacity     float32 `json:"memoryCapacity,omitempty"`
	MemoryThreshold    float32 `json:"memoryThreshold,omitempty"`
	ImageIOCache       bool    `json:"imageIOCache,omitempty"`
	PNGAcceleration    bool    `json:"pngAcceleration,omitempty"`
	JPEGAcceleration   bool    `json:"jpegAcceleration,omitempty"`
	AllowNativeMosaic  bool    `json:"allowNativeMosaic,omitempty"`
	AllowNativeWarp    bool    `json:"allowNativeWarp,omitempty"`
	JAIExt             *JAIExt `json:"jaiext,omitempty"`
}

// JAIExt wraps the JAI-Ext operations list. Same empty-string wire
// quirk as [Contact].
type JAIExt struct {
	JAIExtOperations *JAIExtOperations `json:"jaiExtOperations,omitempty"`
}

// UnmarshalJSON tolerates GeoServer's empty-jaiext wire form.
func (j *JAIExt) UnmarshalJSON(data []byte) error {
	d := bytes.TrimSpace(data)
	if len(d) == 0 || string(d) == "null" || string(d) == `""` {
		return nil
	}
	type alias JAIExt
	return json.Unmarshal(d, (*alias)(j))
}

// JAIExtOperations is the inner operations list.
type JAIExtOperations struct {
	Class  string   `json:"@class,omitempty"`
	String []string `json:"string,omitempty"`
}

// CoverageAccess controls the thread pool that processes raster
// (coverage) access; surfaced under settings.coverageAccess.
type CoverageAccess struct {
	MaxPoolSize           int    `json:"maxPoolSize,omitempty"`
	CorePoolSize          int    `json:"corePoolSize,omitempty"`
	KeepAliveTime         int    `json:"keepAliveTime,omitempty"`
	QueueType             string `json:"queueType,omitempty"`
	ImageIOCacheThreshold int    `json:"imageIOCacheThreshold,omitempty"`
}
