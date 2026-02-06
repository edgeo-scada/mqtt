# Connection Pooling

MQTT connection pool documentation for high-performance applications.

## Overview

The connection pool maintains multiple active MQTT connections and distributes them via round-robin. This enables:

- **High Availability** - Automatic failover if a connection fails
- **High Performance** - Load distribution across connections
- **Resilience** - Health checks and automatic reconnection

## Pool Creation

### NewPool

Creates a new connection pool.

```go
func NewPool(opts ...PoolOption) *Pool
```

**Example:**

```go
pool := mqtt.NewPool(
    mqtt.WithPoolSize(5),
    mqtt.WithPoolHealthCheckInterval(30*time.Second),
    mqtt.WithPoolClientOptions(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithAutoReconnect(true),
    ),
)
```

## Pool Options

### WithPoolSize

Sets the number of connections in the pool.

```go
mqtt.WithPoolSize(size int) PoolOption
```

**Default:** 3

### WithPoolMaxIdleTime

Sets the maximum idle time for a connection.

```go
mqtt.WithPoolMaxIdleTime(d time.Duration) PoolOption
```

**Default:** 5 minutes

### WithPoolHealthCheckInterval

Sets the health check interval.

```go
mqtt.WithPoolHealthCheckInterval(d time.Duration) PoolOption
```

**Default:** 30 seconds

### WithPoolClientOptions

Sets the options for each client in the pool.

```go
mqtt.WithPoolClientOptions(opts ...Option) PoolOption
```

## Methods

### Connect

Initializes all connections in the pool.

```go
func (p *Pool) Connect(ctx context.Context) error
```

**Returns:** `nil` if at least one connection succeeds, otherwise the first error.

**Example:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := pool.Connect(ctx); err != nil {
    log.Fatalf("Pool connection failed: %v", err)
}
```

### Close

Closes all connections in the pool.

```go
func (p *Pool) Close() error
```

### Get

Gets a client from the pool (round-robin).

```go
func (p *Pool) Get() (*Client, error)
```

**Returns:**
- `*Client` - Available MQTT client
- `error` - If no connection is available

**Example:**

```go
client, err := pool.Get()
if err != nil {
    log.Printf("No connection available: %v", err)
    return
}
defer pool.Release(client)

// Use the client...
```

### Release

Returns a client to the pool.

```go
func (p *Pool) Release(client *Client)
```

### Publish

Publishes a message via a pool connection.

```go
func (p *Pool) Publish(ctx context.Context, topic string, payload []byte, qos QoS, retain bool) error
```

**Example:**

```go
if err := pool.Publish(ctx, "sensors/data", payload, mqtt.QoS1, false); err != nil {
    log.Printf("Publish failed: %v", err)
}
```

### Subscribe

Subscribes via a pool connection.

```go
func (p *Pool) Subscribe(ctx context.Context, topic string, qos QoS, handler MessageHandler) error
```

## Pool Metrics

### Metrics

Returns pool metrics.

```go
func (p *Pool) Metrics() *PoolMetrics
```

```go
type PoolMetrics struct {
    TotalClients   Gauge    // Total number of clients
    HealthyClients Gauge    // Number of healthy clients
    TotalRequests  Counter  // Total requests
    FailedRequests Counter  // Failed requests
}
```

### Size

Returns the pool size.

```go
func (p *Pool) Size() int
```

### HealthyCount

Returns the number of healthy connections.

```go
func (p *Pool) HealthyCount() int
```

## Complete Example

### High-Performance Application

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/signal"
    "sync"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

type SensorReading struct {
    SensorID  string  `json:"sensor_id"`
    Value     float64 `json:"value"`
    Timestamp int64   `json:"timestamp"`
}

func main() {
    // Create the pool
    pool := mqtt.NewPool(
        mqtt.WithPoolSize(10),
        mqtt.WithPoolHealthCheckInterval(15*time.Second),
        mqtt.WithPoolMaxIdleTime(2*time.Minute),
        mqtt.WithPoolClientOptions(
            mqtt.WithServer("mqtt://localhost:1883"),
            mqtt.WithKeepAlive(30*time.Second),
            mqtt.WithAutoReconnect(true),
            mqtt.WithConnectRetryInterval(time.Second),
        ),
    )

    // Connect
    ctx := context.Background()
    if err := pool.Connect(ctx); err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer pool.Close()

    log.Printf("Pool connected: %d/%d healthy connections",
        pool.HealthyCount(), pool.Size())

    // Simulate sensors sending data
    var wg sync.WaitGroup
    done := make(chan struct{})

    // 100 simulated sensors
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(sensorID int) {
            defer wg.Done()
            ticker := time.NewTicker(100 * time.Millisecond)
            defer ticker.Stop()

            for {
                select {
                case <-done:
                    return
                case <-ticker.C:
                    reading := SensorReading{
                        SensorID:  fmt.Sprintf("sensor-%03d", sensorID),
                        Value:     20.0 + float64(sensorID%10),
                        Timestamp: time.Now().UnixNano(),
                    }

                    payload, _ := json.Marshal(reading)
                    topic := fmt.Sprintf("sensors/%s/data", reading.SensorID)

                    if err := pool.Publish(ctx, topic, payload, mqtt.QoS0, false); err != nil {
                        log.Printf("Publish error %s: %v", reading.SensorID, err)
                    }
                }
            }
        }(i)
    }

    // Display metrics periodically
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                metrics := pool.Metrics()
                log.Printf("Pool stats: healthy=%d/%d, requests=%d, failed=%d",
                    metrics.HealthyClients.Value(),
                    metrics.TotalClients.Value(),
                    metrics.TotalRequests.Value(),
                    metrics.FailedRequests.Value(),
                )
            }
        }
    }()

    // Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Shutting down...")
    close(done)
    wg.Wait()
}
```

### Load Balancing with Different Handlers

```go
package main

import (
    "context"
    "log"
    "sync"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    pool := mqtt.NewPool(
        mqtt.WithPoolSize(5),
        mqtt.WithPoolClientOptions(
            mqtt.WithServer("mqtt://localhost:1883"),
        ),
    )

    ctx := context.Background()
    if err := pool.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create specialized handlers on different connections
    var wg sync.WaitGroup

    // Temperature data handler
    wg.Add(1)
    go func() {
        defer wg.Done()
        client, err := pool.Get()
        if err != nil {
            log.Printf("Get error: %v", err)
            return
        }
        // Note: Don't Release if keeping the client for subscriptions

        handler := func(c *mqtt.Client, msg *mqtt.Message) {
            log.Printf("Temperature: %s", string(msg.Payload))
        }
        client.Subscribe(ctx, "sensors/+/temperature", mqtt.QoS1, handler)
    }()

    // Humidity data handler
    wg.Add(1)
    go func() {
        defer wg.Done()
        client, err := pool.Get()
        if err != nil {
            log.Printf("Get error: %v", err)
            return
        }

        handler := func(c *mqtt.Client, msg *mqtt.Message) {
            log.Printf("Humidity: %s", string(msg.Payload))
        }
        client.Subscribe(ctx, "sensors/+/humidity", mqtt.QoS1, handler)
    }()

    wg.Wait()
    select {} // Block
}
```

## Best Practices

### 1. Pool Sizing

```go
// General rule: 1 connection per 1000 messages/sec
// Adjust based on network latency and QoS used
mqtt.WithPoolSize(numWorkers / 100)
```

### 2. Error Handling

```go
func publishWithRetry(pool *mqtt.Pool, topic string, payload []byte) error {
    var lastErr error
    for i := 0; i < 3; i++ {
        if err := pool.Publish(context.Background(), topic, payload, mqtt.QoS1, false); err != nil {
            lastErr = err
            time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
            continue
        }
        return nil
    }
    return lastErr
}
```

### 3. Monitoring

```go
// Monitor pool health
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        healthy := pool.HealthyCount()
        total := pool.Size()

        if healthy < total/2 {
            log.Warnf("Pool degraded: %d/%d connections", healthy, total)
            // Alert...
        }
    }
}()
```

### 4. Graceful Shutdown

```go
func shutdown(pool *mqtt.Pool) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Wait for in-flight publications to complete
    // ...

    if err := pool.Close(); err != nil {
        log.Printf("Pool close error: %v", err)
    }
}
```
