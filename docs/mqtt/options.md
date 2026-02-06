# Configuration Options

Complete documentation of MQTT client configuration options.

## Connection Options

### WithServer

Adds an MQTT broker URI.

```go
mqtt.WithServer(uri string) Option
```

**Supported schemes:**
- `mqtt://` or `tcp://` - Standard TCP (port 1883)
- `mqtts://`, `ssl://` or `tls://` - TLS (port 8883)
- `ws://` - WebSocket (port 80)
- `wss://` - WebSocket Secure (port 443)

**Example:**

```go
mqtt.WithServer("mqtt://localhost:1883")
mqtt.WithServer("mqtts://broker.example.com:8883")
mqtt.WithServer("wss://broker.example.com/mqtt")
```

### WithServers

Configures multiple brokers for failover.

```go
mqtt.WithServers(uris ...string) Option
```

**Example:**

```go
mqtt.WithServers(
    "mqtt://broker1.example.com:1883",
    "mqtt://broker2.example.com:1883",
    "mqtt://broker3.example.com:1883",
)
```

### WithClientID

Sets the client identifier.

```go
mqtt.WithClientID(id string) Option
```

**Example:**

```go
mqtt.WithClientID("my-application-001")
```

### WithCleanStart

Enables or disables clean start (new session).

```go
mqtt.WithCleanStart(clean bool) Option
```

- `true` - Starts a new session (default)
- `false` - Resumes existing session if available

### WithKeepAlive

Sets the keep-alive interval.

```go
mqtt.WithKeepAlive(d time.Duration) Option
```

**Default:** 60 seconds

**Example:**

```go
mqtt.WithKeepAlive(30 * time.Second)
```

### WithConnectTimeout

Sets the connection timeout.

```go
mqtt.WithConnectTimeout(d time.Duration) Option
```

**Default:** 30 seconds

### WithWriteTimeout

Sets the write timeout.

```go
mqtt.WithWriteTimeout(d time.Duration) Option
```

**Default:** 10 seconds

## Authentication Options

### WithCredentials

Configures username/password authentication.

```go
mqtt.WithCredentials(username, password string) Option
```

**Example:**

```go
mqtt.WithCredentials("user", "secret123")
```

## TLS Options

### WithTLS

Configures TLS with a custom configuration.

```go
mqtt.WithTLS(config *tls.Config) Option
```

**Example:**

```go
// With client certificate
cert, _ := tls.LoadX509KeyPair("client.crt", "client.key")

mqtt.WithTLS(&tls.Config{
    Certificates:       []tls.Certificate{cert},
    InsecureSkipVerify: false,
    MinVersion:         tls.VersionTLS12,
})
```

### WithTLSInsecure

Disables server certificate verification.

```go
mqtt.WithTLSInsecure() Option
```

> **Warning:** Use only for testing.

## WebSocket Options

### WithWebSocket

Enables WebSocket transport.

```go
mqtt.WithWebSocket(enabled bool) Option
```

### WithWebSocketPath

Sets the WebSocket endpoint path.

```go
mqtt.WithWebSocketPath(path string) Option
```

**Default:** `/mqtt`

**Example:**

```go
mqtt.WithWebSocketPath("/ws/mqtt")
```

## Reconnection Options

### WithAutoReconnect

Enables or disables automatic reconnection.

```go
mqtt.WithAutoReconnect(enabled bool) Option
```

**Default:** true

### WithConnectRetryInterval

Sets the initial retry interval.

```go
mqtt.WithConnectRetryInterval(d time.Duration) Option
```

**Default:** 1 second

### WithMaxReconnectInterval

Sets the maximum retry interval (backoff).

```go
mqtt.WithMaxReconnectInterval(d time.Duration) Option
```

**Default:** 2 minutes

### WithMaxRetries

Sets the maximum number of reconnection attempts.

```go
mqtt.WithMaxRetries(n int) Option
```

**Default:** 0 (unlimited)

## MQTT 5.0 Options

### WithSessionExpiryInterval

Sets the session lifetime in seconds.

```go
mqtt.WithSessionExpiryInterval(seconds uint32) Option
```

- `0` - Session deleted on disconnect
- `0xFFFFFFFF` - Permanent session

**Example:**

```go
mqtt.WithSessionExpiryInterval(3600) // 1 hour
```

### WithReceiveMaximum

Sets the maximum number of QoS 1/2 messages in flight.

```go
mqtt.WithReceiveMaximum(max uint16) Option
```

**Default:** 65535

### WithMaxPacketSize

Sets the maximum accepted packet size.

```go
mqtt.WithMaxPacketSize(size uint32) Option
```

**Default:** 256 MB

### WithTopicAliasMaximum

Sets the maximum number of topic aliases.

```go
mqtt.WithTopicAliasMaximum(max uint16) Option
```

**Default:** 0 (disabled)

### WithRequestResponseInfo

Requests response information from the server.

```go
mqtt.WithRequestResponseInfo(enabled bool) Option
```

### WithRequestProblemInfo

Requests problem information from the server.

```go
mqtt.WithRequestProblemInfo(enabled bool) Option
```

**Default:** true

### WithUserProperties

Sets user properties for CONNECT.

```go
mqtt.WithUserProperties(props []UserProperty) Option
```

**Example:**

```go
mqtt.WithUserProperties([]mqtt.UserProperty{
    {Key: "app-version", Value: "1.0.0"},
    {Key: "client-type", Value: "sensor"},
})
```

## Will Message Options

### WithWill

Configures the Last Will and Testament message.

```go
mqtt.WithWill(topic string, payload []byte, qos QoS, retain bool) Option
```

**Example:**

```go
mqtt.WithWill(
    "devices/sensor-1/status",
    []byte("offline"),
    mqtt.QoS1,
    true,
)
```

### WithWillProperties

Configures Will message properties.

```go
mqtt.WithWillProperties(props *Properties) Option
```

**Example:**

```go
mqtt.WithWillProperties(&mqtt.Properties{
    WillDelayInterval: ptrUint32(60), // 60 second delay
    MessageExpiry:     ptrUint32(3600),
    ContentType:       "application/json",
})
```

## Callbacks

### WithOnConnect

Callback called after successful connection.

```go
mqtt.WithOnConnect(handler OnConnectHandler) Option

type OnConnectHandler func(client *Client)
```

**Example:**

```go
mqtt.WithOnConnect(func(c *mqtt.Client) {
    log.Println("Connected!")
    // Resubscribe to topics
    c.Subscribe(context.Background(), "my/topic", mqtt.QoS1, handler)
})
```

### WithOnConnectionLost

Callback called when connection is lost.

```go
mqtt.WithOnConnectionLost(handler ConnectionLostHandler) Option

type ConnectionLostHandler func(client *Client, err error)
```

**Example:**

```go
mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
    log.Printf("Connection lost: %v", err)
})
```

### WithOnReconnecting

Callback called before each reconnection attempt.

```go
mqtt.WithOnReconnecting(handler ReconnectHandler) Option

type ReconnectHandler func(client *Client, opts *ClientOptions)
```

**Example:**

```go
mqtt.WithOnReconnecting(func(c *mqtt.Client, opts *mqtt.ClientOptions) {
    log.Println("Attempting to reconnect...")
})
```

### WithDefaultMessageHandler

Default handler for messages without a specific handler.

```go
mqtt.WithDefaultMessageHandler(handler MessageHandler) Option

type MessageHandler func(client *Client, msg *Message)
```

## Logging Options

### WithLogger

Configures the logger.

```go
mqtt.WithLogger(logger *slog.Logger) Option
```

**Example:**

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

mqtt.WithLogger(logger)
```

## Complete Example

```go
client := mqtt.NewClient(
    // Connection
    mqtt.WithServer("mqtts://broker.example.com:8883"),
    mqtt.WithClientID("my-iot-device"),
    mqtt.WithCredentials("device", "secret"),

    // TLS
    mqtt.WithTLS(&tls.Config{
        MinVersion: tls.VersionTLS12,
    }),

    // Session
    mqtt.WithCleanStart(false),
    mqtt.WithSessionExpiryInterval(86400), // 24h
    mqtt.WithKeepAlive(30*time.Second),

    // Reconnection
    mqtt.WithAutoReconnect(true),
    mqtt.WithConnectRetryInterval(time.Second),
    mqtt.WithMaxReconnectInterval(time.Minute),

    // MQTT 5.0
    mqtt.WithReceiveMaximum(100),
    mqtt.WithTopicAliasMaximum(10),
    mqtt.WithUserProperties([]mqtt.UserProperty{
        {Key: "device-type", Value: "sensor"},
    }),

    // Will
    mqtt.WithWill("devices/my-device/status", []byte("offline"), mqtt.QoS1, true),

    // Callbacks
    mqtt.WithOnConnect(func(c *mqtt.Client) {
        log.Println("Connected")
    }),
    mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
        log.Printf("Disconnected: %v", err)
    }),

    // Logging
    mqtt.WithLogger(slog.Default()),
)
```
