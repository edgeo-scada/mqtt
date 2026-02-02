// mqttcli is a command-line MQTT client for testing, debugging, and benchmarking.
package main

import (
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
