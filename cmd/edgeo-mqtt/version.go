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

package main

import (
	"fmt"
	"runtime"

	"github.com/edgeo-scada/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var (
	// Build information, set via ldflags
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func printVersion() {
	fmt.Printf("edgeo-mqtt version %s\n", version)
	fmt.Printf("  MQTT Library:  %s\n", mqtt.Version)
	fmt.Printf("  MQTT Protocol: %d (%s)\n", mqtt.ProtocolVersion, mqtt.ProtocolName)
	fmt.Printf("  Go version:    %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Git commit:    %s\n", commit)
	fmt.Printf("  Build date:    %s\n", buildDate)
}
