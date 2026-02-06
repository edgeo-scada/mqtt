# Getting Started

Quick start guide for the MQTT library.

## Prerequisites

- Go 1.22 or higher
- An MQTT broker (Mosquitto, HiveMQ, EMQX, etc.)

## Installation

```bash
go get github.com/edgeo-scada/mqtt
```

## First Client

### Simple Publisher

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    // Create the client with basic options
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-publisher"),
        mqtt.WithCleanStart(true),
    )

    // Connect to the broker
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Disconnect(context.Background())

    log.Println("Connected to MQTT broker")

    // Publish a message
    topic := "test/hello"
    payload := []byte("Hello MQTT!")

    token := client.Publish(context.Background(), topic, payload, mqtt.QoS1, false)
    if err := token.Wait(); err != nil {
        log.Fatalf("Publish failed: %v", err)
    }

    log.Printf("Message published to %s", topic)
}
```

### Simple Subscriber

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    // Handler for received messages
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Message received on %s: %s", msg.Topic, string(msg.Payload))
    }

    // Create the client
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-subscriber"),
        mqtt.WithDefaultMessageHandler(handler),
    )

    // Connect
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Disconnect(context.Background())

    // Subscribe to the topic
    token := client.Subscribe(context.Background(), "test/#", mqtt.QoS1, handler)
    if err := token.Wait(); err != nil {
        log.Fatalf("Subscribe failed: %v", err)
    }
    log.Println("Subscribed to test/#")

    // Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Shutting down...")
}
```

## Secure Connection (TLS)

```go
import "crypto/tls"

client := mqtt.NewClient(
    mqtt.WithServer("mqtts://broker.example.com:8883"),
    mqtt.WithClientID("secure-client"),
    mqtt.WithTLS(&tls.Config{
        // Custom TLS configuration if needed
    }),
)
```

For testing without certificate verification:

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtts://localhost:8883"),
    mqtt.WithTLSInsecure(),
)
```

## WebSocket Connection

```go
// Standard WebSocket
client := mqtt.NewClient(
    mqtt.WithServer("ws://broker.example.com:80"),
    mqtt.WithWebSocketPath("/mqtt"),
)

// Secure WebSocket
client := mqtt.NewClient(
    mqtt.WithServer("wss://broker.example.com:443"),
    mqtt.WithWebSocketPath("/mqtt"),
)
```

## Authentication

### Username/Password

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithCredentials("username", "password"),
)
```

### Client Certificate (mTLS)

```go
cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
if err != nil {
    log.Fatal(err)
}

client := mqtt.NewClient(
    mqtt.WithServer("mqtts://localhost:8883"),
    mqtt.WithTLS(&tls.Config{
        Certificates: []tls.Certificate{cert},
    }),
)
```

## QoS Levels

### QoS 0 - At most once

Message sent once without confirmation. May be lost.

```go
client.Publish(ctx, "topic", payload, mqtt.QoS0, false)
```

### QoS 1 - At least once

Message confirmed by PUBACK. May be duplicated.

```go
token := client.Publish(ctx, "topic", payload, mqtt.QoS1, false)
if err := token.Wait(); err != nil {
    log.Printf("Error: %v", err)
}
```

### QoS 2 - Exactly once

Guaranteed exactly-once delivery (PUBREC/PUBREL/PUBCOMP).

```go
token := client.Publish(ctx, "topic", payload, mqtt.QoS2, false)
if err := token.Wait(); err != nil {
    log.Printf("Error: %v", err)
}
```

## Wildcards

### Single-level wildcard (+)

Matches a single topic level.

```go
// Subscribes to sensors/room1/temperature, sensors/room2/temperature, etc.
client.Subscribe(ctx, "sensors/+/temperature", mqtt.QoS1, handler)
```

### Multi-level wildcard (#)

Matches all remaining levels.

```go
// Subscribes to all topics under sensors/
client.Subscribe(ctx, "sensors/#", mqtt.QoS1, handler)
```

## Will Message (Last Will and Testament)

Configures a message automatically sent if the client disconnects unexpectedly.

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithClientID("sensor-1"),
    mqtt.WithWill(
        "sensors/sensor-1/status",  // topic
        []byte("offline"),           // payload
        mqtt.QoS1,                   // QoS
        true,                        // retain
    ),
)
```

## Automatic Reconnection

By default, the client automatically reconnects with exponential backoff.

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithAutoReconnect(true),
    mqtt.WithConnectRetryInterval(1*time.Second),
    mqtt.WithMaxReconnectInterval(2*time.Minute),
    mqtt.WithMaxRetries(0), // 0 = unlimited
    mqtt.WithOnReconnecting(func(c *mqtt.Client, opts *mqtt.ClientOptions) {
        log.Println("Attempting to reconnect...")
    }),
)
```

## Next Steps

- [Client API](client.md) - Complete client API documentation
- [Options](options.md) - All configuration options
- [Pool](pool.md) - Connection pooling for high performance
- [Metrics](metrics.md) - Monitoring and metrics
