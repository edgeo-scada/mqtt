package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark MQTT operations",
	Long: `Run performance benchmarks on MQTT publish/subscribe operations.

Measures throughput, latency, and error rates for different scenarios.

Sub-commands:
  pub     Benchmark publishing
  sub     Benchmark subscribing
  pubsub  Benchmark round-trip latency

Examples:
  # Benchmark publishing 10000 messages with 10 clients
  edgeo-mqtt bench pub -t "bench" -n 10000 -c 10

  # Benchmark subscribing with 5 clients
  edgeo-mqtt bench sub -t "bench/#" -c 5

  # Measure round-trip latency
  edgeo-mqtt bench pubsub -t "bench" -n 1000`,
}

var benchPubCmd = &cobra.Command{
	Use:   "pub",
	Short: "Benchmark publishing",
	RunE:  runBenchPub,
}

var benchSubCmd = &cobra.Command{
	Use:   "sub",
	Short: "Benchmark subscribing",
	RunE:  runBenchSub,
}

var benchPubSubCmd = &cobra.Command{
	Use:   "pubsub",
	Short: "Benchmark round-trip latency",
	RunE:  runBenchPubSub,
}

var (
	benchTopic      string
	benchCount      int
	benchClients    int
	benchPayload    int
	benchQoS        int
	benchDuration   time.Duration
	benchWarmup     int
)

func init() {
	rootCmd.AddCommand(benchCmd)
	benchCmd.AddCommand(benchPubCmd)
	benchCmd.AddCommand(benchSubCmd)
	benchCmd.AddCommand(benchPubSubCmd)

	// Common flags for all bench subcommands
	for _, cmd := range []*cobra.Command{benchPubCmd, benchSubCmd, benchPubSubCmd} {
		cmd.Flags().StringVarP(&benchTopic, "topic", "t", "edgeo-mqtt/bench", "topic for benchmark")
		cmd.Flags().IntVarP(&benchCount, "count", "n", 10000, "number of messages")
		cmd.Flags().IntVarP(&benchClients, "clients", "c", 1, "number of concurrent clients")
		cmd.Flags().IntVarP(&benchPayload, "payload", "s", 100, "payload size in bytes")
		cmd.Flags().IntVarP(&benchQoS, "qos", "q", 0, "QoS level")
		cmd.Flags().DurationVarP(&benchDuration, "duration", "d", 0, "run for duration instead of count")
		cmd.Flags().IntVar(&benchWarmup, "warmup", 100, "warmup messages (not counted)")
	}
}

type benchResult struct {
	Messages    int64
	Bytes       int64
	Errors      int64
	Duration    time.Duration
	Latencies   []time.Duration
}

func (r *benchResult) MessagesPerSec() float64 {
	return float64(r.Messages) / r.Duration.Seconds()
}

func (r *benchResult) BytesPerSec() float64 {
	return float64(r.Bytes) / r.Duration.Seconds()
}

func (r *benchResult) LatencyStats() (min, avg, max, p99 time.Duration) {
	if len(r.Latencies) == 0 {
		return 0, 0, 0, 0
	}

	sorted := make([]time.Duration, len(r.Latencies))
	copy(sorted, r.Latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	min = sorted[0]
	max = sorted[len(sorted)-1]

	var total time.Duration
	for _, l := range sorted {
		total += l
	}
	avg = total / time.Duration(len(sorted))

	p99Idx := int(float64(len(sorted)) * 0.99)
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}
	p99 = sorted[p99Idx]

	return
}

func runBenchPub(cmd *cobra.Command, args []string) error {
	qos, err := parseQoS(benchQoS)
	if err != nil {
		return err
	}

	// Create payload
	payload := make([]byte, benchPayload)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Printf("Benchmarking publish to %s\n", benchTopic)
	fmt.Printf("  Clients: %d, Messages: %d, Payload: %d bytes, QoS: %d\n\n",
		benchClients, benchCount, benchPayload, qos)

	// Create clients
	clients := make([]*mqtt.Client, benchClients)
	for i := 0; i < benchClients; i++ {
		client, err := createClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create client %d: %w", i, err)
		}
		clients[i] = client
		defer disconnectClient(client)
	}

	// Warmup
	if benchWarmup > 0 {
		fmt.Printf("Warming up (%d messages)...\n", benchWarmup)
		for i := 0; i < benchWarmup; i++ {
			token := clients[0].Publish(ctx, benchTopic, payload, qos, false)
			token.Wait()
		}
	}

	// Run benchmark
	var wg sync.WaitGroup
	var totalMessages atomic.Int64
	var totalBytes atomic.Int64
	var totalErrors atomic.Int64
	allLatencies := make([][]time.Duration, benchClients)

	messagesPerClient := benchCount / benchClients

	fmt.Println("Running benchmark...")
	startTime := time.Now()

	for i := 0; i < benchClients; i++ {
		wg.Add(1)
		go func(clientIdx int) {
			defer wg.Done()

			client := clients[clientIdx]
			var latencies []time.Duration

			for j := 0; j < messagesPerClient; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				msgStart := time.Now()
				token := client.Publish(ctx, fmt.Sprintf("%s/%d", benchTopic, clientIdx), payload, qos, false)
				err := token.Wait()
				latency := time.Since(msgStart)

				if err != nil {
					totalErrors.Add(1)
				} else {
					totalMessages.Add(1)
					totalBytes.Add(int64(len(payload)))
					latencies = append(latencies, latency)
				}
			}

			allLatencies[clientIdx] = latencies
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Collect all latencies
	var allLats []time.Duration
	for _, lats := range allLatencies {
		allLats = append(allLats, lats...)
	}

	result := &benchResult{
		Messages:  totalMessages.Load(),
		Bytes:     totalBytes.Load(),
		Errors:    totalErrors.Load(),
		Duration:  duration,
		Latencies: allLats,
	}

	printBenchResult("Publish", result)
	return nil
}

func runBenchSub(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Printf("Benchmarking subscribe to %s\n", benchTopic)
	fmt.Printf("  Clients: %d, Duration: %s\n\n", benchClients, benchDuration)

	if benchDuration == 0 {
		benchDuration = 10 * time.Second
	}

	// Create subscriber clients
	var wg sync.WaitGroup
	var totalMessages atomic.Int64
	var totalBytes atomic.Int64

	for i := 0; i < benchClients; i++ {
		client, err := createClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create client %d: %w", i, err)
		}
		defer disconnectClient(client)

		handler := func(c *mqtt.Client, msg *mqtt.Message) {
			totalMessages.Add(1)
			totalBytes.Add(int64(len(msg.Payload)))
		}

		token := client.Subscribe(ctx, benchTopic, mqtt.QoS(benchQoS), handler)
		if err := token.Wait(); err != nil {
			return fmt.Errorf("failed to subscribe: %w", err)
		}
	}

	fmt.Printf("Subscribed. Waiting for messages for %s...\n", benchDuration)
	startTime := time.Now()

	select {
	case <-time.After(benchDuration):
	case <-ctx.Done():
	}

	wg.Wait()
	duration := time.Since(startTime)

	result := &benchResult{
		Messages: totalMessages.Load(),
		Bytes:    totalBytes.Load(),
		Duration: duration,
	}

	printBenchResult("Subscribe", result)
	return nil
}

func runBenchPubSub(cmd *cobra.Command, args []string) error {
	qos, err := parseQoS(benchQoS)
	if err != nil {
		return err
	}

	payload := make([]byte, benchPayload)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Printf("Benchmarking round-trip latency on %s\n", benchTopic)
	fmt.Printf("  Messages: %d, Payload: %d bytes, QoS: %d\n\n", benchCount, benchPayload, qos)

	// Create publisher and subscriber
	pubClient, err := createClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}
	defer disconnectClient(pubClient)

	subClient, err := createClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}
	defer disconnectClient(subClient)

	// Setup timing
	responseTopic := benchTopic + "/response"
	var latencies []time.Duration
	var mu sync.Mutex
	received := make(chan struct{}, benchCount)
	timestamps := make(map[string]time.Time)

	// Subscribe to response topic
	handler := func(c *mqtt.Client, msg *mqtt.Message) {
		recvTime := time.Now()
		msgID := string(msg.Payload[:36]) // First 36 chars are the ID

		mu.Lock()
		if sendTime, ok := timestamps[msgID]; ok {
			latencies = append(latencies, recvTime.Sub(sendTime))
			delete(timestamps, msgID)
		}
		mu.Unlock()

		select {
		case received <- struct{}{}:
		default:
		}
	}

	token := subClient.Subscribe(ctx, responseTopic, qos, handler)
	if err := token.Wait(); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Warmup
	if benchWarmup > 0 {
		fmt.Printf("Warming up (%d messages)...\n", benchWarmup)
		for i := 0; i < benchWarmup; i++ {
			msgID := fmt.Sprintf("%036d", i)
			pubClient.Publish(ctx, responseTopic, []byte(msgID+string(payload)), qos, false).Wait()
			select {
			case <-received:
			case <-time.After(time.Second):
			}
		}
		latencies = nil // Clear warmup latencies
	}

	fmt.Println("Running benchmark...")
	startTime := time.Now()

	for i := 0; i < benchCount; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		msgID := fmt.Sprintf("%036d", i+benchWarmup)
		mu.Lock()
		timestamps[msgID] = time.Now()
		mu.Unlock()

		pubClient.Publish(ctx, responseTopic, []byte(msgID+string(payload)), qos, false)
	}

	// Wait for responses
	timeout := time.After(10 * time.Second)
	count := 0
WaitLoop:
	for count < benchCount {
		select {
		case <-received:
			count++
		case <-timeout:
			break WaitLoop
		case <-ctx.Done():
			break WaitLoop
		}
	}

	duration := time.Since(startTime)

	result := &benchResult{
		Messages:  int64(count),
		Bytes:     int64(count * benchPayload),
		Errors:    int64(benchCount - count),
		Duration:  duration,
		Latencies: latencies,
	}

	printBenchResult("Round-trip", result)
	return nil
}

func printBenchResult(name string, result *benchResult) {
	fmt.Println()
	PrintSection(fmt.Sprintf("%s Benchmark Results", name))

	PrintKeyValue("Messages", fmt.Sprintf("%d", result.Messages))
	PrintKeyValue("Errors", fmt.Sprintf("%d", result.Errors))
	PrintKeyValue("Duration", formatDuration(result.Duration))
	PrintKeyValue("Throughput", fmt.Sprintf("%.2f msg/s", result.MessagesPerSec()))
	PrintKeyValue("Bandwidth", fmt.Sprintf("%s/s", formatBytes(int64(result.BytesPerSec()))))

	if len(result.Latencies) > 0 {
		min, avg, max, p99 := result.LatencyStats()
		fmt.Println()
		PrintKeyValue("Latency Min", formatDuration(min))
		PrintKeyValue("Latency Avg", formatDuration(avg))
		PrintKeyValue("Latency Max", formatDuration(max))
		PrintKeyValue("Latency P99", formatDuration(p99))

		// Standard deviation
		if len(result.Latencies) > 1 {
			var sumSquares float64
			avgFloat := float64(avg)
			for _, l := range result.Latencies {
				diff := float64(l) - avgFloat
				sumSquares += diff * diff
			}
			stdDev := time.Duration(math.Sqrt(sumSquares / float64(len(result.Latencies))))
			PrintKeyValue("Latency StdDev", formatDuration(stdDev))
		}
	}

	fmt.Println()
}
