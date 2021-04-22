package geoserver

// CoverageInfo is geoserver CoverageInfo
type Coverage struct {
	Name               string `json:"name,omitempty"`
	NativeCoverageName string `json:"nativeCoverageName,omitempty"`
}
