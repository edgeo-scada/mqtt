package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor topics in real-time",
	Long: `Monitor MQTT topics with real-time updates, change detection, and alerts.

Similar to watch command in modbus/opcua CLIs, this provides a continuously
updated view of message values with statistics.

Examples:
  # Watch topics with live updates
  mqttcli watch -t "sensor/#"

  # Show changes only (diff mode)
  mqttcli watch -t "sensor/temp" --diff

  # Alert on threshold values
  mqttcli watch -t "sensor/temp" --alert-high 30 --alert-low 10

  # Log messages to CSV file
  mqttcli watch -t "data/#" --log messages.csv`,
	RunE: runWatch,
}

var (
	watchTopics    []string
	watchDiff      bool
	watchAlertHigh float64
	watchAlertLow  float64
	watchLogFile   string
	watchInterval  time.Duration
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringArrayVarP(&watchTopics, "topic", "t", nil, "topics to watch (repeatable, required)")
	watchCmd.Flags().BoolVar(&watchDiff, "diff", false, "only show changed values")
	watchCmd.Flags().Float64Var(&watchAlertHigh, "alert-high", 0, "alert when value exceeds threshold")
	watchCmd.Flags().Float64Var(&watchAlertLow, "alert-low", 0, "alert when value below threshold")
	watchCmd.Flags().StringVar(&watchLogFile, "log", "", "log messages to CSV file")
	watchCmd.Flags().DurationVar(&watchInterval, "refresh", 500*time.Millisecond, "screen refresh interval")

	watchCmd.MarkFlagRequired("topic")
}

type watchState struct {
	mu            sync.RWMutex
	topics        map[string]*topicState
	messageCount  int64
	startTime     time.Time
	lastUpdate    time.Time
	logWriter     *csv.Writer
	logFile       *os.File
	alertsEnabled bool
	alertHigh     float64
	alertLow      float64
}

type topicState struct {
	topic      string
	payload    string
	lastValue  string
	qos        mqtt.QoS
	retain     bool
	count      int64
	lastUpdate time.Time
	changed    bool
	numValue   float64
	isNumeric  bool
	alert      string
}

func runWatch(cmd *cobra.Command, args []string) error {
	if len(watchTopics) == 0 {
		return fmt.Errorf("at least one topic is required")
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

	// Initialize state
	state := &watchState{
		topics:        make(map[string]*topicState),
		startTime:     time.Now(),
		alertsEnabled: watchAlertHigh != 0 || watchAlertLow != 0,
		alertHigh:     watchAlertHigh,
		alertLow:      watchAlertLow,
	}

	// Setup logging
	if watchLogFile != "" {
		f, err := os.Create(watchLogFile)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer f.Close()
		state.logFile = f
		state.logWriter = csv.NewWriter(f)
		state.logWriter.Write([]string{"timestamp", "topic", "payload", "qos", "retain"})
	}

	// Message handler
	handler := func(c *mqtt.Client, msg *mqtt.Message) {
		state.handleMessage(msg)
	}

	// Subscribe
	subs := make([]mqtt.Subscription, len(watchTopics))
	for i, topic := range watchTopics {
		subs[i] = mqtt.Subscription{
			Topic: topic,
			QoS:   mqtt.QoS1,
		}
	}

	token := client.SubscribeMultiple(ctx, subs, handler)
	if err := token.Wait(); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	// Clear screen and start display loop
	clearScreen()
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Print("\n")
			state.printSummary()
			return nil
		case <-ticker.C:
			state.render()
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *watchState) handleMessage(msg *mqtt.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageCount++
	s.lastUpdate = time.Now()

	payload := string(msg.Payload)
	topic := msg.Topic

	ts, exists := s.topics[topic]
	if !exists {
		ts = &topicState{
			topic: topic,
		}
		s.topics[topic] = ts
	}

	ts.lastValue = ts.payload
	ts.payload = payload
	ts.qos = msg.QoS
	ts.retain = msg.Retain
	ts.count++
	ts.lastUpdate = time.Now()
	ts.changed = ts.lastValue != payload

	// Try to parse as number for alerts
	if val, err := strconv.ParseFloat(strings.TrimSpace(payload), 64); err == nil {
		ts.numValue = val
		ts.isNumeric = true

		// Check alerts
		ts.alert = ""
		if s.alertsEnabled {
			if s.alertHigh != 0 && val > s.alertHigh {
				ts.alert = "HIGH"
			} else if s.alertLow != 0 && val < s.alertLow {
				ts.alert = "LOW"
			}
		}
	} else {
		ts.isNumeric = false
		ts.alert = ""
	}

	// Log to file
	if s.logWriter != nil {
		s.logWriter.Write([]string{
			time.Now().Format(time.RFC3339Nano),
			topic,
			payload,
			fmt.Sprintf("%d", msg.QoS),
			fmt.Sprintf("%t", msg.Retain),
		})
		s.logWriter.Flush()
	}
}

func (s *watchState) render() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Move cursor to top
	fmt.Print("\033[H")

	// Header
	elapsed := time.Since(s.startTime)
	rate := float64(s.messageCount) / elapsed.Seconds()
	if elapsed < time.Second {
		rate = 0
	}

	fmt.Printf("\033[2K%s MQTT Watch %s\n",
		colorize("●", ColorGreen),
		colorize(fmt.Sprintf("[%s]", formatDuration(elapsed)), ColorGray))

	fmt.Printf("\033[2KTopics: %d | Messages: %d | Rate: %.1f msg/s\n",
		len(s.topics), s.messageCount, rate)

	fmt.Printf("\033[2K%s\n", strings.Repeat("─", 80))

	// Get sorted topics
	topics := make([]*topicState, 0, len(s.topics))
	for _, ts := range s.topics {
		topics = append(topics, ts)
	}
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].topic < topics[j].topic
	})

	// Render each topic
	for _, ts := range topics {
		s.renderTopic(ts)
	}

	// Clear remaining lines
	fmt.Print("\033[J")

	// Footer
	if watchLogFile != "" {
		fmt.Printf("\n%s Logging to: %s\n", colorize("●", ColorBlue), watchLogFile)
	}
	fmt.Printf("\nPress Ctrl+C to exit\n")
}

func (s *watchState) renderTopic(ts *topicState) {
	// Skip unchanged in diff mode
	if watchDiff && !ts.changed {
		return
	}

	// Build line
	var sb strings.Builder

	// Topic name
	sb.WriteString(colorize(ts.topic, ColorCyan))
	sb.WriteString(" ")

	// QoS indicator
	sb.WriteString(colorize(fmt.Sprintf("[Q%d]", ts.qos), ColorGray))
	sb.WriteString(" ")

	// Retain indicator
	if ts.retain {
		sb.WriteString(colorize("[R]", ColorMagenta))
		sb.WriteString(" ")
	}

	// Change indicator
	if ts.changed {
		sb.WriteString(colorize("*", ColorYellow))
		sb.WriteString(" ")
	}

	// Alert
	if ts.alert != "" {
		if ts.alert == "HIGH" {
			sb.WriteString(colorize("[!HIGH]", ColorRed))
		} else {
			sb.WriteString(colorize("[!LOW]", ColorBlue))
		}
		sb.WriteString(" ")
	}

	// Payload (truncate if too long)
	payload := ts.payload
	if len(payload) > 50 {
		payload = payload[:50] + "..."
	}
	// Escape newlines
	payload = strings.ReplaceAll(payload, "\n", "\\n")
	payload = strings.ReplaceAll(payload, "\r", "\\r")
	sb.WriteString(payload)

	// Count and age
	age := time.Since(ts.lastUpdate)
	ageStr := formatDuration(age)
	sb.WriteString(colorize(fmt.Sprintf(" (%d, %s ago)", ts.count, ageStr), ColorGray))

	fmt.Printf("\033[2K%s\n", sb.String())
}

func (s *watchState) printSummary() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	elapsed := time.Since(s.startTime)

	PrintSection("Watch Summary")
	PrintKeyValue("Duration", formatDuration(elapsed))
	PrintKeyValue("Topics", fmt.Sprintf("%d", len(s.topics)))
	PrintKeyValue("Messages", fmt.Sprintf("%d", s.messageCount))
	PrintKeyValue("Rate", fmt.Sprintf("%.2f msg/s", float64(s.messageCount)/elapsed.Seconds()))

	if watchLogFile != "" {
		PrintKeyValue("Log file", watchLogFile)
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}
