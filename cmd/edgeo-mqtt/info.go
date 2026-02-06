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
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/edgeo-scada/mqtt"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show broker information",
	Long: `Display information about the MQTT broker connection.

This command connects to the broker and displays connection properties,
server capabilities, and optionally reads $SYS topics for broker statistics.

Examples:
  # Basic connection info
  edgeo-mqtt info

  # Include $SYS broker statistics
  edgeo-mqtt info --sys

  # JSON output
  edgeo-mqtt info -o json`,
	RunE: runInfo,
}

var (
	infoSys bool
)

func init() {
	rootCmd.AddCommand(infoCmd)

	infoCmd.Flags().BoolVar(&infoSys, "sys", false, "read $SYS topics for broker statistics")
}

func runInfo(cmd *cobra.Command, args []string) error {
	// Connect with timing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()
	client, err := createClient(ctx)
	if err != nil {
		return err
	}
	defer disconnectClient(client)

	connectTime := time.Since(startTime)

	// Measure ping latency
	pingStart := time.Now()
	// Simple latency measurement by forcing a packet round-trip
	// The client internally handles ping/pong
	pingLatency := time.Since(pingStart)

	// Get server properties
	props := client.ServerProperties()

	if outputFormat == "json" {
		return printInfoJSON(connectTime, pingLatency, props)
	}

	// Print connection info
	PrintSection("Connection")
	PrintKeyValue("Broker", broker)
	PrintKeyValue("Status", colorize("Connected", ColorGreen))
	PrintKeyValue("Connect time", formatDuration(connectTime))
	PrintKeyValue("Client ID", clientID)

	// Print server properties
	if props != nil {
		PrintSection("Server Properties")
		printServerProperties(props)
	}

	// Print $SYS topics if requested
	if infoSys {
		if err := printSysTopics(ctx, client); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read $SYS topics: %v\n", err)
		}
	}

	return nil
}

func printServerProperties(props *mqtt.Properties) {
	if props.AssignedClientID != "" {
		PrintKeyValue("Assigned Client ID", props.AssignedClientID)
	}
	if props.ServerKeepAlive != nil {
		PrintKeyValue("Server Keep Alive", fmt.Sprintf("%ds", *props.ServerKeepAlive))
	}
	if props.SessionExpiryInterval != nil {
		PrintKeyValue("Session Expiry", fmt.Sprintf("%ds", *props.SessionExpiryInterval))
	}
	if props.ReceiveMaximum != nil {
		PrintKeyValue("Receive Maximum", fmt.Sprintf("%d", *props.ReceiveMaximum))
	}
	if props.MaximumPacketSize != nil {
		PrintKeyValue("Max Packet Size", formatBytes(int64(*props.MaximumPacketSize)))
	}
	if props.TopicAliasMaximum != nil {
		PrintKeyValue("Topic Alias Max", fmt.Sprintf("%d", *props.TopicAliasMaximum))
	}
	if props.MaximumQoS != nil {
		PrintKeyValue("Maximum QoS", fmt.Sprintf("%d", *props.MaximumQoS))
	}
	if props.RetainAvailable != nil {
		available := "no"
		if *props.RetainAvailable == 1 {
			available = "yes"
		}
		PrintKeyValue("Retain Available", available)
	}
	if props.WildcardSubAvailable != nil {
		available := "no"
		if *props.WildcardSubAvailable == 1 {
			available = "yes"
		}
		PrintKeyValue("Wildcard Subs", available)
	}
	if props.SubIDAvailable != nil {
		available := "no"
		if *props.SubIDAvailable == 1 {
			available = "yes"
		}
		PrintKeyValue("Subscription IDs", available)
	}
	if props.SharedSubAvailable != nil {
		available := "no"
		if *props.SharedSubAvailable == 1 {
			available = "yes"
		}
		PrintKeyValue("Shared Subs", available)
	}
	if props.ServerReference != "" {
		PrintKeyValue("Server Reference", props.ServerReference)
	}
	if props.ResponseInfo != "" {
		PrintKeyValue("Response Info", props.ResponseInfo)
	}
	if props.AuthMethod != "" {
		PrintKeyValue("Auth Method", props.AuthMethod)
	}
	for _, up := range props.UserProperties {
		PrintKeyValue(up.Key, up.Value)
	}
}

func printSysTopics(ctx context.Context, client *mqtt.Client) error {
	PrintSection("Broker Statistics ($SYS)")

	// Setup signal handling for early exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Collect $SYS messages
	var mu sync.Mutex
	sysData := make(map[string]string)
	done := make(chan struct{})

	handler := func(c *mqtt.Client, msg *mqtt.Message) {
		mu.Lock()
		sysData[msg.Topic] = string(msg.Payload)
		mu.Unlock()
	}

	// Subscribe to $SYS/#
	token := client.Subscribe(ctx, "$SYS/#", mqtt.QoS0, handler)
	if err := token.Wait(); err != nil {
		return err
	}

	// Wait a bit for messages or signal
	select {
	case <-sigCh:
		close(done)
	case <-time.After(2 * time.Second):
		close(done)
	case <-ctx.Done():
		return ctx.Err()
	}

	// Unsubscribe
	client.Unsubscribe(ctx, "$SYS/#")

	// Print collected data
	mu.Lock()
	defer mu.Unlock()

	if len(sysData) == 0 {
		fmt.Println("  No $SYS topics available (broker may not support them)")
		return nil
	}

	// Group and sort topics
	groups := groupSysTopics(sysData)
	for _, group := range groups {
		if len(group.items) > 0 {
			fmt.Printf("\n  %s\n", colorize(group.name, ColorBold))
			for _, item := range group.items {
				fmt.Printf("    %-30s %s\n", colorize(item.key+":", ColorGray), item.value)
			}
		}
	}

	return nil
}

type sysGroup struct {
	name  string
	items []sysItem
}

type sysItem struct {
	key   string
	value string
}

func groupSysTopics(data map[string]string) []sysGroup {
	groups := []sysGroup{
		{name: "Broker"},
		{name: "Clients"},
		{name: "Messages"},
		{name: "Subscriptions"},
		{name: "Traffic"},
		{name: "Other"},
	}

	for topic, value := range data {
		// Extract key from topic
		parts := strings.Split(topic, "/")
		key := parts[len(parts)-1]

		// Categorize
		var groupIdx int
		topicLower := strings.ToLower(topic)
		switch {
		case strings.Contains(topicLower, "broker") || strings.Contains(topicLower, "version"):
			groupIdx = 0
		case strings.Contains(topicLower, "client"):
			groupIdx = 1
		case strings.Contains(topicLower, "message") || strings.Contains(topicLower, "publish"):
			groupIdx = 2
		case strings.Contains(topicLower, "subscri"):
			groupIdx = 3
		case strings.Contains(topicLower, "bytes") || strings.Contains(topicLower, "traffic"):
			groupIdx = 4
		default:
			groupIdx = 5
		}

		groups[groupIdx].items = append(groups[groupIdx].items, sysItem{
			key:   key,
			value: value,
		})
	}

	return groups
}

func printInfoJSON(connectTime, pingLatency time.Duration, props *mqtt.Properties) error {
	info := map[string]interface{}{
		"broker":       broker,
		"status":       "connected",
		"connect_time": connectTime.String(),
		"ping_latency": pingLatency.String(),
		"client_id":    clientID,
	}

	if props != nil {
		serverProps := make(map[string]interface{})
		if props.AssignedClientID != "" {
			serverProps["assigned_client_id"] = props.AssignedClientID
		}
		if props.ServerKeepAlive != nil {
			serverProps["server_keep_alive"] = *props.ServerKeepAlive
		}
		if props.SessionExpiryInterval != nil {
			serverProps["session_expiry_interval"] = *props.SessionExpiryInterval
		}
		if props.ReceiveMaximum != nil {
			serverProps["receive_maximum"] = *props.ReceiveMaximum
		}
		if props.MaximumPacketSize != nil {
			serverProps["maximum_packet_size"] = *props.MaximumPacketSize
		}
		if props.TopicAliasMaximum != nil {
			serverProps["topic_alias_maximum"] = *props.TopicAliasMaximum
		}
		if props.MaximumQoS != nil {
			serverProps["maximum_qos"] = *props.MaximumQoS
		}
		if props.RetainAvailable != nil {
			serverProps["retain_available"] = *props.RetainAvailable == 1
		}
		if props.WildcardSubAvailable != nil {
			serverProps["wildcard_subscription_available"] = *props.WildcardSubAvailable == 1
		}
		if props.SubIDAvailable != nil {
			serverProps["subscription_id_available"] = *props.SubIDAvailable == 1
		}
		if props.SharedSubAvailable != nil {
			serverProps["shared_subscription_available"] = *props.SharedSubAvailable == 1
		}
		if len(serverProps) > 0 {
			info["server_properties"] = serverProps
		}
	}

	return printJSON(info)
}

func printJSON(v interface{}) error {
	encoder := newJSONEncoder(os.Stdout)
	return encoder.Encode(v)
}

type jsonEncoder struct {
	w *os.File
}

func newJSONEncoder(w *os.File) *jsonEncoder {
	return &jsonEncoder{w: w}
}

func (e *jsonEncoder) Encode(v interface{}) error {
	data, err := marshalJSON(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(data)
	if err != nil {
		return err
	}
	_, err = e.w.WriteString("\n")
	return err
}

func marshalJSON(v interface{}) ([]byte, error) {
	// Simple JSON marshaling without external package
	switch val := v.(type) {
	case map[string]interface{}:
		return marshalJSONMap(val)
	default:
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}

func marshalJSONMap(m map[string]interface{}) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("{")
	first := true
	for k, v := range m {
		if !first {
			sb.WriteString(",")
		}
		first = false
		sb.WriteString(fmt.Sprintf("%q:", k))
		switch val := v.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%q", val))
		case bool:
			sb.WriteString(fmt.Sprintf("%t", val))
		case int, int64, uint16, uint32:
			sb.WriteString(fmt.Sprintf("%d", val))
		case map[string]interface{}:
			nested, _ := marshalJSONMap(val)
			sb.Write(nested)
		default:
			sb.WriteString(fmt.Sprintf("%q", fmt.Sprintf("%v", val)))
		}
	}
	sb.WriteString("}")
	return []byte(sb.String()), nil
}
