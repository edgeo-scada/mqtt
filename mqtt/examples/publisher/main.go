// Example MQTT publisher demonstrating basic publish functionality.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
	// Create client with options
	client := mqtt.NewClient(
		mqtt.WithServer("mqtt://localhost:1883"),
		mqtt.WithClientID("example-publisher"),
		mqtt.WithCleanStart(true),
		mqtt.WithKeepAlive(30*time.Second),
		mqtt.WithAutoReconnect(true),
		mqtt.WithOnConnect(func(c *mqtt.Client) {
			log.Println("Connected to broker")
		}),
		mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
			log.Printf("Connection lost: %v", err)
		}),
	)

	// Connect to broker
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Publish messages periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-sigCh:
			log.Println("Shutting down...")
			return
		case <-ticker.C:
			count++
			topic := "test/messages"
			payload := fmt.Sprintf("Hello MQTT! Message #%d", count)

			token := client.Publish(context.Background(), topic, []byte(payload), mqtt.QoS1, false)
			if err := token.Wait(); err != nil {
				log.Printf("Failed to publish: %v", err)
				continue
			}

			log.Printf("Published message #%d to %s", count, topic)
		}
	}
}
