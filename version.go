// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
