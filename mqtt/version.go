// Package mqtt provides a pure Go MQTT 5.0 client implementation.
package mqtt

// Version information for the MQTT client library.
const (
	// Version is the current version of the library.
	Version = "1.0.0"

	// ProtocolName is the MQTT protocol name.
	ProtocolName = "MQTT"

	// ProtocolVersion is the MQTT 5.0 protocol version.
	ProtocolVersion = 5
)

// BuildInfo contains build metadata.
type BuildInfo struct {
	Version   string
	GoVersion string
	OS        string
	Arch      string
	BuildTime string
}

// GetBuildInfo returns the current build information.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version: Version,
	}
}
