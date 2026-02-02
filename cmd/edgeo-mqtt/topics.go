package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var topicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "Discover topics",
	Long: `Discover and list MQTT topics.

This command subscribes to topics (using wildcards) and collects
unique topics seen during the observation period.

Examples:
  # Discover all topics by listening for 10 seconds
  edgeo-mqtt topics -t "#" -d 10s

  # List $SYS broker topics
  edgeo-mqtt topics --sys

  # Watch specific topic pattern
  edgeo-mqtt topics -t "sensor/#" -d 30s`,
	RunE: runTopics,
}

var (
	topicsPattern  []string
	topicsDuration time.Duration
	topicsSys      bool
)

func init() {
	rootCmd.AddCommand(topicsCmd)

	topicsCmd.Flags().StringArrayVarP(&topicsPattern, "topic", "t", []string{"#"}, "topic pattern to subscribe to")
	topicsCmd.Flags().DurationVarP(&topicsDuration, "duration", "d", 5*time.Second, "discovery duration")
	topicsCmd.Flags().BoolVar(&topicsSys, "sys", false, "discover $SYS topics")
}

func runTopics(cmd *cobra.Command, args []string) error {
	// If --sys is set, override pattern
	if topicsSys {
		topicsPattern = []string{"$SYS/#"}
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

	// Collect topics
	var mu sync.Mutex
	discovered := make(map[string]*topicInfo)

	handler := func(c *mqtt.Client, msg *mqtt.Message) {
		mu.Lock()
		defer mu.Unlock()

		topic := msg.Topic
		if info, exists := discovered[topic]; exists {
			info.count++
			info.lastSeen = time.Now()
			info.lastPayloadLen = len(msg.Payload)
		} else {
			discovered[topic] = &topicInfo{
				topic:          topic,
				count:          1,
				firstSeen:      time.Now(),
				lastSeen:       time.Now(),
				lastPayloadLen: len(msg.Payload),
				qos:            msg.QoS,
				retain:         msg.Retain,
			}
		}
	}

	// Subscribe
	subs := make([]mqtt.Subscription, len(topicsPattern))
	for i, pattern := range topicsPattern {
		subs[i] = mqtt.Subscription{
			Topic: pattern,
			QoS:   mqtt.QoS0,
		}
	}

	token := client.SubscribeMultiple(ctx, subs, handler)
	if err := token.Wait(); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Discovering topics for %s...\n", topicsDuration)

	// Wait for duration or signal
	select {
	case <-sigCh:
		fmt.Fprintln(os.Stderr)
	case <-time.After(topicsDuration):
	case <-ctx.Done():
		return nil
	}

	// Unsubscribe
	for _, pattern := range topicsPattern {
		client.Unsubscribe(ctx, pattern)
	}

	// Print results
	mu.Lock()
	defer mu.Unlock()

	if len(discovered) == 0 {
		fmt.Println("No topics discovered")
		return nil
	}

	// Sort topics
	topics := make([]*topicInfo, 0, len(discovered))
	for _, info := range discovered {
		topics = append(topics, info)
	}
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].topic < topics[j].topic
	})

	// Output
	switch outputFormat {
	case "json":
		printTopicsJSON(topics)
	case "csv":
		printTopicsCSV(topics)
	default:
		printTopicsTable(topics)
	}

	return nil
}

type topicInfo struct {
	topic          string
	count          int64
	firstSeen      time.Time
	lastSeen       time.Time
	lastPayloadLen int
	qos            mqtt.QoS
	retain         bool
}

func printTopicsTable(topics []*topicInfo) {
	fmt.Printf("\n%s\n", colorize(fmt.Sprintf("Discovered %d topic(s)", len(topics)), ColorBold))
	fmt.Println(strings.Repeat("-", 80))

	// Group by prefix for better readability
	groups := groupTopicsByPrefix(topics)

	for _, group := range groups {
		if group.prefix != "" {
			fmt.Printf("\n%s\n", colorize(group.prefix, ColorCyan))
		}
		for _, t := range group.topics {
			displayTopic := t.topic
			if group.prefix != "" {
				displayTopic = "  " + strings.TrimPrefix(t.topic, group.prefix)
			}

			flags := ""
			if t.retain {
				flags = colorize("[R]", ColorMagenta)
			}

			fmt.Printf("%-50s %5d msgs  %s  %s\n",
				displayTopic,
				t.count,
				colorize(fmt.Sprintf("~%d bytes", t.lastPayloadLen), ColorGray),
				flags)
		}
	}

	// Summary
	var totalMsgs int64
	for _, t := range topics {
		totalMsgs += t.count
	}
	fmt.Printf("\nTotal: %d topic(s), %d message(s)\n", len(topics), totalMsgs)
}

func printTopicsJSON(topics []*topicInfo) {
	fmt.Println("[")
	for i, t := range topics {
		comma := ","
		if i == len(topics)-1 {
			comma = ""
		}
		fmt.Printf(`  {"topic":%q,"count":%d,"last_payload_size":%d,"retain":%t}%s`+"\n",
			t.topic, t.count, t.lastPayloadLen, t.retain, comma)
	}
	fmt.Println("]")
}

func printTopicsCSV(topics []*topicInfo) {
	fmt.Println("topic,count,last_payload_size,retain")
	for _, t := range topics {
		fmt.Printf("%s,%d,%d,%t\n", t.topic, t.count, t.lastPayloadLen, t.retain)
	}
}

type topicGroup struct {
	prefix string
	topics []*topicInfo
}

func groupTopicsByPrefix(topics []*topicInfo) []topicGroup {
	// Find common prefixes
	prefixes := make(map[string][]*topicInfo)

	for _, t := range topics {
		parts := strings.Split(t.topic, "/")
		if len(parts) > 1 {
			prefix := parts[0] + "/"
			prefixes[prefix] = append(prefixes[prefix], t)
		} else {
			prefixes[""] = append(prefixes[""], t)
		}
	}

	// Convert to slice and sort
	var groups []topicGroup
	for prefix, tList := range prefixes {
		groups = append(groups, topicGroup{
			prefix: prefix,
			topics: tList,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].prefix == "" {
			return false
		}
		if groups[j].prefix == "" {
			return true
		}
		return groups[i].prefix < groups[j].prefix
	})

	return groups
}
