package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var pubCmd = &cobra.Command{
	Use:   "pub",
	Short: "Publish a message to a topic",
	Long: `Publish one or more messages to an MQTT topic.

The message payload can be provided via:
  - The -m/--message flag
  - A file with -f/--file
  - Standard input (piped)

Examples:
  # Simple publish
  edgeo-mqtt pub -t "sensor/temp" -m "23.5"

  # Publish with QoS 1 and retain
  edgeo-mqtt pub -t "config/device1" -m '{"enabled":true}' -q 1 --retain

  # Publish from file
  edgeo-mqtt pub -t "data/batch" -f payload.json

  # Publish from stdin
  echo "hello" | edgeo-mqtt pub -t "test"

  # Repeated publishing (100 messages, 1 second interval)
  edgeo-mqtt pub -t "heartbeat" -m "ping" -n 100 -i 1s

  # MQTT 5.0 properties
  edgeo-mqtt pub -t "request/123" -m "data" \
    --response-topic "response/123" \
    --correlation-data "req-001" \
    --content-type "application/json"`,
	RunE: runPub,
}

var (
	pubTopic           string
	pubMessage         string
	pubFile            string
	pubQoS             int
	pubRetain          bool
	pubCount           int
	pubInterval        time.Duration
	pubContentType     string
	pubResponseTopic   string
	pubCorrelationData string
	pubUserProps       []string
	pubPayloadFormat   int
	pubMessageExpiry   uint32
)

func init() {
	rootCmd.AddCommand(pubCmd)

	pubCmd.Flags().StringVarP(&pubTopic, "topic", "t", "", "topic to publish to (required)")
	pubCmd.Flags().StringVarP(&pubMessage, "message", "m", "", "message payload")
	pubCmd.Flags().StringVarP(&pubFile, "file", "f", "", "read payload from file")
	pubCmd.Flags().IntVarP(&pubQoS, "qos", "q", 0, "QoS level (0, 1, or 2)")
	pubCmd.Flags().BoolVarP(&pubRetain, "retain", "r", false, "retain message")
	pubCmd.Flags().IntVarP(&pubCount, "count", "n", 1, "number of messages to publish (0 = infinite)")
	pubCmd.Flags().DurationVarP(&pubInterval, "interval", "i", 0, "interval between messages")
	pubCmd.Flags().StringVar(&pubContentType, "content-type", "", "content type (MQTT 5.0)")
	pubCmd.Flags().StringVar(&pubResponseTopic, "response-topic", "", "response topic (MQTT 5.0)")
	pubCmd.Flags().StringVar(&pubCorrelationData, "correlation-data", "", "correlation data (MQTT 5.0)")
	pubCmd.Flags().StringArrayVar(&pubUserProps, "user-prop", nil, "user property key=value (MQTT 5.0, repeatable)")
	pubCmd.Flags().IntVar(&pubPayloadFormat, "payload-format", -1, "payload format indicator: 0=bytes, 1=UTF-8 (MQTT 5.0)")
	pubCmd.Flags().Uint32Var(&pubMessageExpiry, "message-expiry", 0, "message expiry interval in seconds (MQTT 5.0)")

	pubCmd.MarkFlagRequired("topic")
}

func runPub(cmd *cobra.Command, args []string) error {
	// Get payload
	payload, err := getPayload()
	if err != nil {
		return err
	}

	// Parse QoS
	qos, err := parseQoS(pubQoS)
	if err != nil {
		return err
	}

	// Build properties
	props := buildPublishProperties()

	// Connect
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := createClient(ctx)
	if err != nil {
		return err
	}
	defer disconnectClient(client)

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Publish messages
	count := pubCount
	if count == 0 {
		count = -1 // infinite
	}

	published := 0
	startTime := time.Now()

	for i := 0; count < 0 || i < count; i++ {
		select {
		case <-ctx.Done():
			printVerbose("Interrupted, published %d messages", published)
			return nil
		default:
		}

		// Format payload with counter if template
		msgPayload := formatPayload(payload, i+1)

		var token *mqtt.PublishToken
		if props != nil {
			token = client.PublishWithProperties(ctx, pubTopic, msgPayload, qos, pubRetain, props)
		} else {
			token = client.Publish(ctx, pubTopic, msgPayload, qos, pubRetain)
		}

		if err := token.Wait(); err != nil {
			return fmt.Errorf("publish failed: %w", err)
		}

		published++

		if verbose {
			fmt.Printf("Published [%d] to %s (QoS %d, Retain: %v): %s\n",
				published, pubTopic, qos, pubRetain, truncateString(string(msgPayload), 100))
		}

		// Wait for interval
		if pubInterval > 0 && (count < 0 || i < count-1) {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(pubInterval):
			}
		}
	}

	elapsed := time.Since(startTime)
	if !verbose {
		fmt.Printf("Published %d message(s) to %s in %s\n", published, pubTopic, formatDuration(elapsed))
	} else {
		rate := float64(published) / elapsed.Seconds()
		fmt.Printf("Published %d message(s) in %s (%.2f msg/s)\n", published, formatDuration(elapsed), rate)
	}

	return nil
}

func getPayload() ([]byte, error) {
	// From message flag
	if pubMessage != "" {
		return []byte(pubMessage), nil
	}

	// From file
	if pubFile != "" {
		data, err := os.ReadFile(pubFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		return data, nil
	}

	// From stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		// Trim trailing newline
		if len(data) > 0 && data[len(data)-1] == '\n' {
			data = data[:len(data)-1]
		}
		return data, nil
	}

	return nil, fmt.Errorf("message is required (-m, -f, or stdin)")
}

func buildPublishProperties() *mqtt.Properties {
	hasProps := pubContentType != "" ||
		pubResponseTopic != "" ||
		pubCorrelationData != "" ||
		len(pubUserProps) > 0 ||
		pubPayloadFormat >= 0 ||
		pubMessageExpiry > 0

	if !hasProps {
		return nil
	}

	props := &mqtt.Properties{
		ContentType:   pubContentType,
		ResponseTopic: pubResponseTopic,
	}

	if pubCorrelationData != "" {
		props.CorrelationData = []byte(pubCorrelationData)
	}

	if len(pubUserProps) > 0 {
		props.UserProperties = parseUserProperties(pubUserProps)
	}

	if pubPayloadFormat >= 0 {
		format := byte(pubPayloadFormat)
		props.PayloadFormat = &format
	}

	if pubMessageExpiry > 0 {
		props.MessageExpiry = &pubMessageExpiry
	}

	return props
}

func formatPayload(payload []byte, counter int) []byte {
	// Simple template substitution for {n} pattern
	s := string(payload)
	result := ""
	i := 0
	for i < len(s) {
		if i+2 < len(s) && s[i:i+3] == "{n}" {
			result += fmt.Sprintf("%d", counter)
			i += 3
		} else {
			result += string(s[i])
			i++
		}
	}
	return []byte(result)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
