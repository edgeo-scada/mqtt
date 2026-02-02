package main

import (
	"fmt"
	"runtime"

	"github.com/edgeo/drivers/mqtt/mqtt"
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
	fmt.Printf("mqttcli version %s\n", version)
	fmt.Printf("  MQTT Library:  %s\n", mqtt.Version)
	fmt.Printf("  MQTT Protocol: %d (%s)\n", mqtt.ProtocolVersion, mqtt.ProtocolName)
	fmt.Printf("  Go version:    %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Git commit:    %s\n", commit)
	fmt.Printf("  Build date:    %s\n", buildDate)
}
