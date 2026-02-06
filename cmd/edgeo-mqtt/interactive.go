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
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/edgeo-scada/mqtt/mqtt"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"repl", "shell"},
	Short:   "Interactive MQTT shell",
	Long: `Start an interactive MQTT shell session.

The interactive mode provides a REPL (Read-Eval-Print Loop) for
working with MQTT topics interactively.

Commands:
  connect [broker]      Connect to broker (or reconnect)
  disconnect            Disconnect from broker
  status                Show connection status

  pub <topic> <msg>     Publish message (optional: qos, retain)
  sub <topic>           Subscribe to topic (optional: qos)
  unsub <topic>         Unsubscribe from topic

  topics                List active subscriptions
  clear                 Clear screen
  help                  Show help
  quit, exit            Exit interactive mode

Examples:
  edgeo-mqtt interactive
  edgeo-mqtt interactive -b mqtt://broker:1883`,
	RunE: runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

type interactiveSession struct {
	client        *mqtt.Client
	broker        string
	connected     bool
	subscriptions map[string]mqtt.QoS
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	msgCount      int64
}

func runInteractive(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := &interactiveSession{
		broker:        broker,
		subscriptions: make(map[string]mqtt.QoS),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nUse 'quit' or 'exit' to leave interactive mode")
	}()

	// Auto-connect if broker specified
	if broker != "" {
		if err := session.connect(broker); err != nil {
			fmt.Printf("Warning: could not connect to %s: %v\n", broker, err)
		}
	}

	// Print welcome message
	fmt.Println(colorize("MQTT Interactive Shell", ColorBold))
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	// REPL loop
	reader := bufio.NewReader(os.Stdin)
	for {
		prompt := session.getPrompt()
		fmt.Print(prompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if shouldExit := session.execute(line); shouldExit {
			break
		}
	}

	// Cleanup
	if session.connected {
		session.disconnect()
	}

	return nil
}

func (s *interactiveSession) getPrompt() string {
	status := colorize("disconnected", ColorRed)
	host := "mqtt"

	if s.connected {
		status = colorize("connected", ColorGreen)
		// Extract host from broker URL
		if idx := strings.Index(s.broker, "://"); idx >= 0 {
			host = s.broker[idx+3:]
			if colonIdx := strings.Index(host, ":"); colonIdx >= 0 {
				host = host[:colonIdx]
			}
		}
	}

	return fmt.Sprintf("mqtt[%s@%s]> ", status, host)
}

func (s *interactiveSession) execute(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "quit", "exit", "q":
		fmt.Println("Goodbye!")
		return true

	case "help", "?":
		s.showHelp()

	case "clear", "cls":
		clearScreen()

	case "connect":
		brokerArg := s.broker
		if len(args) > 0 {
			brokerArg = args[0]
		}
		if err := s.connect(brokerArg); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

	case "disconnect":
		s.disconnect()

	case "status":
		s.showStatus()

	case "pub", "publish":
		s.publish(args)

	case "sub", "subscribe":
		s.subscribe(args)

	case "unsub", "unsubscribe":
		s.unsubscribe(args)

	case "topics", "subs":
		s.listSubscriptions()

	default:
		fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
	}

	return false
}

func (s *interactiveSession) showHelp() {
	fmt.Println(`
Available commands:

  Connection:
    connect [broker]      Connect to broker
    disconnect            Disconnect from broker
    status                Show connection status

  Pub/Sub:
    pub <topic> <msg> [qos] [retain]   Publish message
    sub <topic> [qos]                  Subscribe to topic
    unsub <topic>                      Unsubscribe from topic
    topics                             List active subscriptions

  Other:
    clear                 Clear screen
    help                  Show this help
    quit, exit            Exit interactive mode

Examples:
    pub sensor/temp 23.5
    pub config/device {"enabled":true} 1 true
    sub sensor/#
    sub events 1
    unsub sensor/#
`)
}

func (s *interactiveSession) connect(brokerURL string) error {
	if s.connected {
		s.disconnect()
	}

	s.broker = brokerURL
	broker = brokerURL // Update global for createClient

	client, err := createClient(s.ctx)
	if err != nil {
		return err
	}

	s.client = client
	s.connected = true
	fmt.Printf("Connected to %s\n", brokerURL)

	return nil
}

func (s *interactiveSession) disconnect() {
	if !s.connected {
		fmt.Println("Not connected")
		return
	}

	disconnectClient(s.client)
	s.connected = false
	s.subscriptions = make(map[string]mqtt.QoS)
	fmt.Println("Disconnected")
}

func (s *interactiveSession) showStatus() {
	fmt.Println()
	if s.connected {
		PrintKeyValue("Status", colorize("Connected", ColorGreen))
		PrintKeyValue("Broker", s.broker)
		PrintKeyValue("Subscriptions", fmt.Sprintf("%d", len(s.subscriptions)))
		PrintKeyValue("Messages received", fmt.Sprintf("%d", s.msgCount))

		if props := s.client.ServerProperties(); props != nil {
			if props.ServerKeepAlive != nil {
				PrintKeyValue("Keep Alive", fmt.Sprintf("%ds", *props.ServerKeepAlive))
			}
		}
	} else {
		PrintKeyValue("Status", colorize("Disconnected", ColorRed))
	}
	fmt.Println()
}

func (s *interactiveSession) publish(args []string) {
	if !s.connected {
		fmt.Println("Not connected. Use 'connect' first.")
		return
	}

	if len(args) < 2 {
		fmt.Println("Usage: pub <topic> <message> [qos] [retain]")
		return
	}

	topic := args[0]
	message := args[1]

	// Parse optional QoS
	qos := mqtt.QoS0
	if len(args) > 2 {
		q, err := strconv.Atoi(args[2])
		if err == nil && q >= 0 && q <= 2 {
			qos = mqtt.QoS(q)
		}
	}

	// Parse optional retain
	retain := false
	if len(args) > 3 {
		retain = args[3] == "true" || args[3] == "1"
	}

	token := s.client.Publish(s.ctx, topic, []byte(message), qos, retain)
	if err := token.Wait(); err != nil {
		fmt.Printf("Publish failed: %v\n", err)
		return
	}

	fmt.Printf("Published to %s (QoS %d, Retain: %v)\n", topic, qos, retain)
}

func (s *interactiveSession) subscribe(args []string) {
	if !s.connected {
		fmt.Println("Not connected. Use 'connect' first.")
		return
	}

	if len(args) < 1 {
		fmt.Println("Usage: sub <topic> [qos]")
		return
	}

	topic := args[0]

	// Parse optional QoS
	qos := mqtt.QoS0
	if len(args) > 1 {
		q, err := strconv.Atoi(args[1])
		if err == nil && q >= 0 && q <= 2 {
			qos = mqtt.QoS(q)
		}
	}

	// Create handler that prints messages
	handler := func(c *mqtt.Client, msg *mqtt.Message) {
		s.mu.Lock()
		s.msgCount++
		s.mu.Unlock()

		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("\n%s %s %s %s\n",
			colorize(timestamp, ColorGray),
			colorize(msg.Topic, ColorCyan),
			colorize(fmt.Sprintf("[Q%d]", msg.QoS), ColorYellow),
			string(msg.Payload))
		fmt.Print(s.getPrompt())
	}

	token := s.client.Subscribe(s.ctx, topic, qos, handler)
	if err := token.Wait(); err != nil {
		fmt.Printf("Subscribe failed: %v\n", err)
		return
	}

	s.mu.Lock()
	s.subscriptions[topic] = qos
	s.mu.Unlock()

	fmt.Printf("Subscribed to %s with QoS %d\n", topic, qos)
}

func (s *interactiveSession) unsubscribe(args []string) {
	if !s.connected {
		fmt.Println("Not connected. Use 'connect' first.")
		return
	}

	if len(args) < 1 {
		fmt.Println("Usage: unsub <topic>")
		return
	}

	topic := args[0]

	token := s.client.Unsubscribe(s.ctx, topic)
	if err := token.Wait(); err != nil {
		fmt.Printf("Unsubscribe failed: %v\n", err)
		return
	}

	s.mu.Lock()
	delete(s.subscriptions, topic)
	s.mu.Unlock()

	fmt.Printf("Unsubscribed from %s\n", topic)
}

func (s *interactiveSession) listSubscriptions() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.subscriptions) == 0 {
		fmt.Println("No active subscriptions")
		return
	}

	fmt.Println("\nActive subscriptions:")
	for topic, qos := range s.subscriptions {
		fmt.Printf("  %s (QoS %d)\n", colorize(topic, ColorCyan), qos)
	}
	fmt.Println()
}
