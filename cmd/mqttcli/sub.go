package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var subCmd = &cobra.Command{
	Use:   "sub",
	Short: "Subscribe to topics",
	Long: `Subscribe to one or more MQTT topics and print received messages.

Supports MQTT wildcards:
  - + matches a single level (e.g., sensor/+/temp)
  - # matches multiple levels (e.g., sensor/#)

Examples:
  # Subscribe to a single topic
  mqttcli sub -t "sensor/temp"

  # Subscribe to multiple topics
  mqttcli sub -t "sensor/#" -t "actuator/#"

  # Limit to 10 messages
  mqttcli sub -t "events" -c 10

  # JSON output with timestamps
  mqttcli sub -t "data" -o json --timestamps

  # Show MQTT 5.0 properties
  mqttcli sub -t "request/#" --show-properties`,
	RunE: runSub,
}

var (
	subTopics         []string
	subQoS            int
	subCount          int
	subNoLocal        bool
	subRetainHandling int
	subTimestamps     bool
	subShowProps      bool
)

func init() {
	rootCmd.AddCommand(subCmd)

	subCmd.Flags().StringArrayVarP(&subTopics, "topic", "t", nil, "topics to subscribe to (repeatable, required)")
	subCmd.Flags().IntVarP(&subQoS, "qos", "q", 0, "QoS level (0, 1, or 2)")
	subCmd.Flags().IntVarP(&subCount, "count", "c", 0, "maximum number of messages (0 = unlimited)")
	subCmd.Flags().BoolVar(&subNoLocal, "no-local", false, "don't receive own messages (MQTT 5.0)")
	subCmd.Flags().IntVar(&subRetainHandling, "retain-handling", 0, "retain handling: 0=send, 1=new only, 2=none")
	subCmd.Flags().BoolVar(&subTimestamps, "timestamps", false, "show timestamps")
	subCmd.Flags().BoolVar(&subShowProps, "show-properties", false, "show MQTT 5.0 properties")

	subCmd.MarkFlagRequired("topic")
}

func runSub(cmd *cobra.Command, args []string) error {
	if len(subTopics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}

	// Parse QoS
	qos, err := parseQoS(subQoS)
	if err != nil {
		return err
	}

	// Validate retain handling
	if subRetainHandling < 0 || subRetainHandling > 2 {
		return fmt.Errorf("retain-handling must be 0, 1, or 2")
	}

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

	// Message counter
	var msgCount atomic.Int64
	done := make(chan struct{})

	// Create formatter
	formatter := NewFormatter(outputFormat, subTimestamps, subShowProps)

	// Message handler
	handler := func(c *mqtt.Client, msg *mqtt.Message) {
		formatter.FormatMessage(msg)
		msgCount.Add(1)

		if subCount > 0 && msgCount.Load() >= int64(subCount) {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	}

	// Build subscriptions
	subs := make([]mqtt.Subscription, len(subTopics))
	for i, topic := range subTopics {
		subs[i] = mqtt.Subscription{
			Topic:          topic,
			QoS:            qos,
			NoLocal:        subNoLocal,
			RetainHandling: mqtt.RetainHandling(subRetainHandling),
		}
	}

	// Subscribe
	token := client.SubscribeMultiple(ctx, subs, handler)
	if err := token.Wait(); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	printVerbose("Subscribed to %d topic(s) with QoS %d", len(subTopics), qos)

	// Print header
	if verbose {
		for _, topic := range subTopics {
			fmt.Fprintf(os.Stderr, "  - %s\n", topic)
		}
		fmt.Fprintln(os.Stderr, "Waiting for messages... (Ctrl+C to exit)")
	}

	// Wait for messages or signal
	startTime := time.Now()
	select {
	case <-sigCh:
		fmt.Fprintln(os.Stderr)
	case <-done:
	case <-ctx.Done():
	}

	elapsed := time.Since(startTime)
	received := msgCount.Load()

	if verbose {
		rate := float64(received) / elapsed.Seconds()
		fmt.Fprintf(os.Stderr, "Received %d message(s) in %s (%.2f msg/s)\n",
			received, formatDuration(elapsed), rate)
	}

	// Unsubscribe
	unsubToken := client.Unsubscribe(ctx, subTopics...)
	unsubToken.WaitTimeout(5 * time.Second)

	return nil
}
