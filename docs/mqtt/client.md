# Client API

Complete documentation of the MQTT client API.

## Client Creation

### NewClient

Creates a new MQTT client instance.

```go
func NewClient(opts ...Option) *Client
```

**Example:**

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithClientID("my-client"),
    mqtt.WithKeepAlive(60*time.Second),
)
```

## Connection Methods

### Connect

Establishes a connection to the MQTT broker.

```go
func (c *Client) Connect(ctx context.Context) error
```

**Parameters:**
- `ctx` - Context for cancellation and timeout

**Returns:**
- `error` - Connection error or nil

**Example:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := client.Connect(ctx); err != nil {
    log.Fatalf("Connection failed: %v", err)
}
```

### Disconnect

Gracefully closes the connection to the broker.

```go
func (c *Client) Disconnect(ctx context.Context) error
```

**Example:**

```go
if err := client.Disconnect(context.Background()); err != nil {
    log.Printf("Error during disconnect: %v", err)
}
```

## Publishing

### Publish

Publishes a message to a topic.

```go
func (c *Client) Publish(ctx context.Context, topic string, payload []byte, qos QoS, retain bool) *PublishToken
```

**Parameters:**
- `ctx` - Context for cancellation
- `topic` - Destination topic
- `payload` - Message content
- `qos` - QoS level (0, 1, or 2)
- `retain` - If true, message is retained by the broker

**Returns:**
- `*PublishToken` - Token to track the asynchronous operation

**Example:**

```go
token := client.Publish(ctx, "sensors/temp", []byte("22.5"), mqtt.QoS1, false)
if err := token.Wait(); err != nil {
    log.Printf("Publish error: %v", err)
}
```

### PublishWithProperties

Publishes a message with MQTT 5.0 properties.

```go
func (c *Client) PublishWithProperties(ctx context.Context, topic string, payload []byte, qos QoS, retain bool, props *Properties) *PublishToken
```

**Example:**

```go
props := &mqtt.Properties{
    ContentType:     "application/json",
    MessageExpiry:   ptrUint32(3600), // Expires after 1 hour
    ResponseTopic:   "response/topic",
    CorrelationData: []byte("request-123"),
    UserProperties: []mqtt.UserProperty{
        {Key: "source", Value: "sensor-1"},
    },
}

token := client.PublishWithProperties(ctx, "requests/data", payload, mqtt.QoS1, false, props)
```

## Subscription

### Subscribe

Subscribes to a topic with a handler.

```go
func (c *Client) Subscribe(ctx context.Context, topic string, qos QoS, handler MessageHandler) *SubscribeToken
```

**Parameters:**
- `ctx` - Context for cancellation
- `topic` - Topic filter (may contain wildcards)
- `qos` - Maximum QoS requested
- `handler` - Function called for each message

**Example:**

```go
handler := func(c *mqtt.Client, msg *mqtt.Message) {
    log.Printf("Received on %s: %s", msg.Topic, string(msg.Payload))
}

token := client.Subscribe(ctx, "sensors/#", mqtt.QoS1, handler)
if err := token.Wait(); err != nil {
    log.Printf("Subscribe error: %v", err)
}

// Check granted QoS
log.Printf("Granted QoS: %v", token.GrantedQoS)
```

### SubscribeMultiple

Subscribes to multiple topics in a single request.

```go
func (c *Client) SubscribeMultiple(ctx context.Context, subs []Subscription, handler MessageHandler) *SubscribeToken
```

**Example:**

```go
subs := []mqtt.Subscription{
    {Topic: "sensors/temperature", QoS: mqtt.QoS1},
    {Topic: "sensors/humidity", QoS: mqtt.QoS1},
    {Topic: "alerts/#", QoS: mqtt.QoS2, NoLocal: true},
}

token := client.SubscribeMultiple(ctx, subs, handler)
if err := token.Wait(); err != nil {
    log.Printf("Error: %v", err)
}
```

### Unsubscribe

Unsubscribes from one or more topics.

```go
func (c *Client) Unsubscribe(ctx context.Context, topics ...string) *UnsubscribeToken
```

**Example:**

```go
token := client.Unsubscribe(ctx, "sensors/temperature", "sensors/humidity")
if err := token.Wait(); err != nil {
    log.Printf("Error: %v", err)
}
```

## Connection State

### IsConnected

Checks if the client is connected.

```go
func (c *Client) IsConnected() bool
```

### State

Returns the current connection state.

```go
func (c *Client) State() ConnectionState
```

**Possible states:**
- `StateDisconnected` - Not connected
- `StateConnecting` - Connection in progress
- `StateConnected` - Connected
- `StateDisconnecting` - Disconnection in progress

**Example:**

```go
switch client.State() {
case mqtt.StateConnected:
    log.Println("Client connected")
case mqtt.StateConnecting:
    log.Println("Connection in progress...")
case mqtt.StateDisconnected:
    log.Println("Client disconnected")
}
```

## Server Properties

### ServerProperties

Returns properties sent by the server in CONNACK.

```go
func (c *Client) ServerProperties() *Properties
```

**Example:**

```go
props := client.ServerProperties()
if props != nil {
    if props.MaximumQoS != nil {
        log.Printf("Maximum supported QoS: %d", *props.MaximumQoS)
    }
    if props.TopicAliasMaximum != nil {
        log.Printf("Topic alias max: %d", *props.TopicAliasMaximum)
    }
    if props.RetainAvailable != nil && *props.RetainAvailable == 0 {
        log.Println("Retain not supported by broker")
    }
}
```

## Metrics

### Metrics

Returns client metrics.

```go
func (c *Client) Metrics() *Metrics
```

**Example:**

```go
metrics := client.Metrics()
snapshot := metrics.Snapshot()

log.Printf("Messages sent: %d", snapshot.MessagesSent)
log.Printf("Messages received: %d", snapshot.MessagesReceived)
log.Printf("Reconnect attempts: %d", snapshot.ReconnectAttempts)
```

## Tokens

Asynchronous operations return tokens to track their state.

### Token interface

```go
type Token interface {
    Wait() error
    WaitTimeout(timeout time.Duration) error
    Done() <-chan struct{}
    Error() error
}
```

### Waiting with timeout

```go
token := client.Publish(ctx, topic, payload, mqtt.QoS1, false)
if err := token.WaitTimeout(5 * time.Second); err != nil {
    if err == mqtt.ErrTimeout {
        log.Println("Publish timeout")
    } else {
        log.Printf("Error: %v", err)
    }
}
```

### Using with select

```go
token := client.Subscribe(ctx, topic, mqtt.QoS1, handler)

select {
case <-token.Done():
    if err := token.Error(); err != nil {
        log.Printf("Error: %v", err)
    } else {
        log.Println("Subscribe successful")
    }
case <-time.After(10 * time.Second):
    log.Println("Timeout")
case <-ctx.Done():
    log.Println("Cancelled")
}
```

## Types

### Message

Structure representing a received message.

```go
type Message struct {
    Topic      string       // Message topic
    Payload    []byte       // Message content
    QoS        QoS          // QoS level
    Retain     bool         // Retain flag
    PacketID   uint16       // Packet ID (QoS > 0)
    Properties *Properties  // MQTT 5.0 properties
    Duplicate  bool         // Duplicate flag
}
```

### MessageHandler

Message handler signature.

```go
type MessageHandler func(client *Client, msg *Message)
```

### Subscription

Subscription options.

```go
type Subscription struct {
    Topic             string         // Topic filter
    QoS               QoS            // Requested QoS
    NoLocal           bool           // Don't receive own messages
    RetainAsPublished bool           // Keep original retain flag
    RetainHandling    RetainHandling // Retained message handling
}
```

## Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

type SensorData struct {
    DeviceID    string  `json:"device_id"`
    Temperature float64 `json:"temperature"`
    Humidity    float64 `json:"humidity"`
    Timestamp   int64   `json:"timestamp"`
}

func main() {
    // Client configuration
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("sensor-gateway"),
        mqtt.WithCleanStart(false),
        mqtt.WithSessionExpiryInterval(3600),
        mqtt.WithKeepAlive(30*time.Second),
        mqtt.WithAutoReconnect(true),
        mqtt.WithWill("gateways/sensor-gateway/status", []byte("offline"), mqtt.QoS1, true),
        mqtt.WithOnConnect(func(c *mqtt.Client) {
            log.Println("Connected to broker")
            // Publish online status
            c.Publish(context.Background(), "gateways/sensor-gateway/status", []byte("online"), mqtt.QoS1, true)
        }),
        mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
            log.Printf("Connection lost: %v", err)
        }),
    )

    // Connect
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Command handler
    commandHandler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Command received: %s", string(msg.Payload))

        // Reply if response topic is specified
        if msg.Properties != nil && msg.Properties.ResponseTopic != "" {
            response := []byte(`{"status":"ok"}`)
            c.PublishWithProperties(ctx, msg.Properties.ResponseTopic, response, mqtt.QoS1, false,
                &mqtt.Properties{
                    CorrelationData: msg.Properties.CorrelationData,
                })
        }
    }

    // Subscribe to commands
    client.Subscribe(ctx, "gateways/sensor-gateway/commands", mqtt.QoS1, commandHandler)

    // Publish data periodically
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)

    for {
        select {
        case <-sigCh:
            log.Println("Shutting down...")
            return
        case <-ticker.C:
            data := SensorData{
                DeviceID:    "sensor-gateway",
                Temperature: 22.5,
                Humidity:    65.0,
                Timestamp:   time.Now().Unix(),
            }

            payload, _ := json.Marshal(data)
            token := client.Publish(ctx, "sensors/data", payload, mqtt.QoS1, false)
            if err := token.Wait(); err != nil {
                log.Printf("Publish error: %v", err)
            }
        }
    }
}
```
