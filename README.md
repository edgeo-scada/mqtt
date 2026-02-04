# MQTT Client Library

A pure Go MQTT 5.0 client library with support for TLS, WebSocket, and connection pooling.

## Features

- **MQTT 5.0 Protocol**: Full support for MQTT 5.0 features including properties, reason codes, and topic aliases
- **Multiple Transport Options**: TCP, TLS/SSL, WebSocket (WS), and Secure WebSocket (WSS)
- **Connection Pooling**: Built-in connection pool for high-throughput applications
- **Automatic Reconnection**: Configurable auto-reconnect with exponential backoff
- **QoS Support**: All QoS levels (0, 1, 2) with proper message flow handling
- **Topic Wildcards**: Full support for `+` and `#` wildcards in subscriptions
- **Authentication**: Username/password and certificate-based authentication
- **Metrics**: Built-in metrics for monitoring connection and message statistics
- **Functional Options**: Clean, extensible configuration pattern

## Installation

```bash
go get github.com/edgeo-scada/mqtt
```

## Quick Start

### Publisher

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-publisher"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    token := client.Publish(ctx, "test/topic", []byte("Hello MQTT!"), mqtt.QoS1, false)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }
}
```

### Subscriber

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Received: %s -> %s", msg.Topic, string(msg.Payload))
    }

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-subscriber"),
        mqtt.WithDefaultMessageHandler(handler),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    token := client.Subscribe(ctx, "test/#", mqtt.QoS1, handler)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
}
```

## Configuration Options

### Connection Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithServer(uri)` | Broker URI (mqtt://, mqtts://, ws://, wss://) | Required |
| `WithClientID(id)` | Client identifier | Auto-generated |
| `WithCredentials(user, pass)` | Authentication credentials | None |
| `WithCleanStart(bool)` | Clean session on connect | true |
| `WithKeepAlive(duration)` | Keep-alive interval | 60s |
| `WithConnectTimeout(duration)` | Connection timeout | 30s |
| `WithAutoReconnect(bool)` | Enable auto-reconnect | true |
| `WithMaxReconnectInterval(duration)` | Max reconnect backoff | 2m |

### TLS Options

| Option | Description |
|--------|-------------|
| `WithTLS(config)` | Custom TLS configuration |
| `WithTLSInsecure()` | Skip certificate verification |

### WebSocket Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithWebSocket(bool)` | Enable WebSocket transport | false |
| `WithWebSocketPath(path)` | WebSocket endpoint path | /mqtt |

### Will Message

```go
mqtt.WithWill("status/client", []byte("offline"), mqtt.QoS1, true)
```

### Callbacks

```go
mqtt.WithOnConnect(func(c *mqtt.Client) {
    log.Println("Connected")
})

mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
    log.Printf("Disconnected: %v", err)
})

mqtt.WithOnReconnecting(func(c *mqtt.Client, opts *mqtt.ClientOptions) {
    log.Println("Reconnecting...")
})
```

## Connection Pooling

```go
pool := mqtt.NewPool(
    mqtt.WithPoolSize(5),
    mqtt.WithPoolHealthCheckInterval(30*time.Second),
    mqtt.WithPoolClientOptions(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithAutoReconnect(true),
    ),
)

ctx := context.Background()
if err := pool.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Publish using pooled connection
if err := pool.Publish(ctx, "topic", []byte("message"), mqtt.QoS1, false); err != nil {
    log.Printf("Publish failed: %v", err)
}
```

## MQTT 5.0 Properties

```go
props := &mqtt.Properties{
    ContentType:    "application/json",
    ResponseTopic:  "response/topic",
    CorrelationData: []byte("request-123"),
    UserProperties: []mqtt.UserProperty{
        {Key: "custom-key", Value: "custom-value"},
    },
}

client.PublishWithProperties(ctx, "topic", payload, mqtt.QoS1, false, props)
```

## CLI Tool (edgeo-mqtt)

A complete MQTT 5.0 command-line client for testing, debugging, monitoring, and benchmarking.

### Installation

```bash
go build -o edgeo-mqtt ./cmd/edgeo-mqtt
```

### Commands

| Command | Description |
|---------|-------------|
| `pub` | Publish messages to a topic |
| `sub` | Subscribe to topics |
| `watch` | Monitor topics in real-time |
| `info` | Show broker information |
| `topics` | Discover topics |
| `interactive` | Interactive REPL shell |
| `bench` | Performance benchmarks |
| `version` | Print version information |

### Global Flags

```
-b, --broker string      MQTT broker URI (default "mqtt://localhost:1883")
-u, --username string    Username for authentication
-P, --password string    Password for authentication
    --client-id string   Client ID (auto-generated if empty)
    --ca-file string     CA certificate file
    --cert-file string   Client certificate file
    --key-file string    Client key file
    --insecure           Skip TLS certificate verification
    --timeout duration   Connection timeout (default 30s)
    --keepalive duration Keep-alive interval (default 1m)
-o, --output string      Output format: table, json, csv, raw (default "table")
-v, --verbose            Verbose output
    --config string      Config file (default ~/.edgeo-mqtt.yaml)
```

### Publish Examples

```bash
# Simple publish
edgeo-mqtt pub -t "sensor/temp" -m "23.5"

# Publish with QoS 1 and retain
edgeo-mqtt pub -t "config/device" -m '{"enabled":true}' -q 1 --retain

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
  --content-type "application/json"
```

### Subscribe Examples

```bash
# Subscribe to a single topic
edgeo-mqtt sub -t "sensor/temp"

# Subscribe with wildcards
edgeo-mqtt sub -t "sensor/#"

# Multiple topics
edgeo-mqtt sub -t "sensor/#" -t "actuator/#"

# Limit to 10 messages
edgeo-mqtt sub -t "events" -c 10

# JSON output with timestamps
edgeo-mqtt sub -t "data" -o json --timestamps
```

### Watch Mode (Real-time Monitoring)

```bash
# Watch topics with live updates
edgeo-mqtt watch -t "sensor/#"

# Show only changed values
edgeo-mqtt watch -t "sensor/temp" --diff

# Alert on threshold values
edgeo-mqtt watch -t "sensor/temp" --alert-high 30 --alert-low 10

# Log messages to CSV file
edgeo-mqtt watch -t "data/#" --log messages.csv
```

### Interactive Mode

```bash
edgeo-mqtt interactive -b mqtt://localhost:1883
```

Commands in interactive mode:
- `connect [broker]` - Connect to broker
- `disconnect` - Disconnect
- `status` - Show connection status
- `pub <topic> <message> [qos] [retain]` - Publish
- `sub <topic> [qos]` - Subscribe
- `unsub <topic>` - Unsubscribe
- `topics` - List active subscriptions
- `help` - Show help
- `quit` - Exit

### Benchmarking

```bash
# Benchmark publishing (10000 messages, 10 clients)
edgeo-mqtt bench pub -t "bench" -n 10000 -c 10

# Benchmark subscribing (5 clients)
edgeo-mqtt bench sub -t "bench/#" -c 5

# Measure round-trip latency
edgeo-mqtt bench pubsub -t "bench" -n 1000
```

### Configuration File

Create `~/.edgeo-mqtt.yaml`:

```yaml
broker: mqtt://localhost:1883
client-id: my-client
username: user
password: secret
timeout: 30s
keepalive: 60s
output: table
```

## Metrics

```go
metrics := client.Metrics()
snapshot := metrics.Snapshot()

fmt.Printf("Messages sent: %d\n", snapshot.MessagesSent)
fmt.Printf("Messages received: %d\n", snapshot.MessagesReceived)
fmt.Printf("Connection attempts: %d\n", snapshot.ConnectionAttempts)
fmt.Printf("Uptime: %v\n", snapshot.Uptime)
```

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `Connect(ctx)` | Connect to broker |
| `Disconnect(ctx)` | Gracefully disconnect |
| `Publish(ctx, topic, payload, qos, retain)` | Publish message |
| `PublishWithProperties(...)` | Publish with MQTT 5.0 properties |
| `Subscribe(ctx, topic, qos, handler)` | Subscribe to topic |
| `SubscribeMultiple(ctx, subs, handler)` | Subscribe to multiple topics |
| `Unsubscribe(ctx, topics...)` | Unsubscribe from topics |
| `IsConnected()` | Check connection status |
| `State()` | Get connection state |
| `Metrics()` | Get metrics |
| `ServerProperties()` | Get server CONNACK properties |

### QoS Levels

| Level | Name | Description |
|-------|------|-------------|
| 0 | At most once | Fire and forget |
| 1 | At least once | Acknowledged delivery |
| 2 | Exactly once | Assured delivery |

## License

MIT License
