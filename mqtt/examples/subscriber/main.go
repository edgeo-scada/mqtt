// Example MQTT subscriber demonstrating subscription functionality.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
	// Message handler
	messageHandler := func(client *mqtt.Client, msg *mqtt.Message) {
		log.Printf("Received message on %s: %s", msg.Topic, string(msg.Payload))
	}

	// Create client with options
	client := mqtt.NewClient(
		mqtt.WithServer("mqtt://localhost:1883"),
		mqtt.WithClientID("example-subscriber"),
		mqtt.WithCleanStart(true),
		mqtt.WithKeepAlive(30*time.Second),
		mqtt.WithAutoReconnect(true),
		mqtt.WithDefaultMessageHandler(messageHandler),
		mqtt.WithOnConnect(func(c *mqtt.Client) {
			log.Println("Connected to broker")

			// Resubscribe on reconnect
			token := c.Subscribe(context.Background(), "test/#", mqtt.QoS1, messageHandler)
			if err := token.Wait(); err != nil {
				log.Printf("Failed to subscribe: %v", err)
			} else {
				log.Println("Subscribed to test/#")
			}
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

	// Subscribe to topics
	token := client.Subscribe(context.Background(), "test/#", mqtt.QoS1, messageHandler)
	if err := token.Wait(); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	log.Println("Subscribed to test/#")

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
}
