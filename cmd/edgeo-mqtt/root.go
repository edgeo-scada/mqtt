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
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	cfgFile   string
	broker    string
	clientID  string
	username  string
	password  string
	caFile    string
	certFile  string
	keyFile   string
	insecure  bool
	timeout   time.Duration
	keepAlive time.Duration

	// Output flags
	outputFormat string
	verbose      bool
	noColor      bool
)

var rootCmd = &cobra.Command{
	Use:   "edgeo-mqtt",
	Short: "MQTT command-line client",
	Long: `edgeo-mqtt is a complete MQTT 5.0 command-line client for testing,
debugging, monitoring, and benchmarking MQTT brokers.

Supports multiple transport protocols:
  - mqtt://  - Plain TCP connection
  - mqtts:// - TLS encrypted connection
  - ws://    - WebSocket connection
  - wss://   - Secure WebSocket connection

Examples:
  # Publish a message
  edgeo-mqtt pub -t "sensor/temp" -m "23.5"

  # Subscribe to a topic
  edgeo-mqtt sub -t "sensor/#"

  # Interactive mode
  edgeo-mqtt interactive

  # Benchmark publishing
  edgeo-mqtt bench pub -t "bench" -n 10000`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Connection flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "", "", "config file (default is $HOME/.edgeo-mqtt.yaml)")
	rootCmd.PersistentFlags().StringVarP(&broker, "broker", "b", "mqtt://localhost:1883", "MQTT broker URI")
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "client ID (auto-generated if empty)")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "username for authentication")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "P", "", "password for authentication")
	rootCmd.PersistentFlags().StringVar(&caFile, "ca-file", "", "CA certificate file")
	rootCmd.PersistentFlags().StringVar(&certFile, "cert-file", "", "client certificate file")
	rootCmd.PersistentFlags().StringVar(&keyFile, "key-file", "", "client key file")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "connection timeout")
	rootCmd.PersistentFlags().DurationVar(&keepAlive, "keepalive", 60*time.Second, "keep-alive interval")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table, json, csv, raw")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	// Bind flags to viper
	viper.BindPFlag("broker", rootCmd.PersistentFlags().Lookup("broker"))
	viper.BindPFlag("client-id", rootCmd.PersistentFlags().Lookup("client-id"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("ca-file", rootCmd.PersistentFlags().Lookup("ca-file"))
	viper.BindPFlag("cert-file", rootCmd.PersistentFlags().Lookup("cert-file"))
	viper.BindPFlag("key-file", rootCmd.PersistentFlags().Lookup("key-file"))
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("keepalive", rootCmd.PersistentFlags().Lookup("keepalive"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		viper.AddConfigPath(home)
		viper.AddConfigPath(filepath.Join(home, ".config"))
		viper.SetConfigName(".edgeo-mqtt")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("EDGEO_MQTT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Apply viper values to flags
	broker = viper.GetString("broker")
	clientID = viper.GetString("client-id")
	username = viper.GetString("username")
	password = viper.GetString("password")
	caFile = viper.GetString("ca-file")
	certFile = viper.GetString("cert-file")
	keyFile = viper.GetString("key-file")
	insecure = viper.GetBool("insecure")
	timeout = viper.GetDuration("timeout")
	keepAlive = viper.GetDuration("keepalive")
	outputFormat = viper.GetString("output")
	verbose = viper.GetBool("verbose")
	noColor = viper.GetBool("no-color")
}
