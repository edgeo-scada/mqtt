# Exemple Publisher

Exemples de publication de messages MQTT.

## Publisher basique

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
        mqtt.WithClientID("basic-publisher"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Publication simple
    token := client.Publish(ctx, "test/topic", []byte("Hello!"), mqtt.QoS1, false)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }

    log.Println("Message publié")
}
```

## Publisher périodique

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("periodic-publisher"),
        mqtt.WithAutoReconnect(true),
        mqtt.WithOnConnect(func(c *mqtt.Client) {
            log.Println("Connecté")
        }),
        mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
            log.Printf("Déconnecté: %v", err)
        }),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Publication toutes les secondes
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)

    count := 0
    for {
        select {
        case <-sigCh:
            log.Println("Arrêt...")
            return
        case <-ticker.C:
            count++
            payload := fmt.Sprintf("Message #%d at %s", count, time.Now().Format(time.RFC3339))

            token := client.Publish(ctx, "test/messages", []byte(payload), mqtt.QoS1, false)
            if err := token.Wait(); err != nil {
                log.Printf("Erreur: %v", err)
                continue
            }
            log.Printf("Publié: %s", payload)
        }
    }
}
```

## Publisher JSON

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "math/rand"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

type SensorData struct {
    DeviceID    string  `json:"device_id"`
    Temperature float64 `json:"temperature"`
    Humidity    float64 `json:"humidity"`
    Timestamp   string  `json:"timestamp"`
}

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("sensor-publisher"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)

    for {
        select {
        case <-sigCh:
            return
        case <-ticker.C:
            data := SensorData{
                DeviceID:    "sensor-001",
                Temperature: 20.0 + rand.Float64()*10,
                Humidity:    40.0 + rand.Float64()*40,
                Timestamp:   time.Now().Format(time.RFC3339),
            }

            payload, _ := json.Marshal(data)
            token := client.Publish(ctx, "sensors/sensor-001/data", payload, mqtt.QoS1, false)
            if err := token.Wait(); err != nil {
                log.Printf("Erreur: %v", err)
            }
        }
    }
}
```

## Publisher avec propriétés MQTT 5.0

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
        mqtt.WithClientID("mqtt5-publisher"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Propriétés MQTT 5.0
    msgExpiry := uint32(3600) // 1 heure
    payloadFormat := byte(1) // UTF-8

    props := &mqtt.Properties{
        PayloadFormat:   &payloadFormat,
        MessageExpiry:   &msgExpiry,
        ContentType:     "application/json",
        ResponseTopic:   "responses/my-request",
        CorrelationData: []byte("request-123"),
        UserProperties: []mqtt.UserProperty{
            {Key: "source", Value: "sensor-gateway"},
            {Key: "priority", Value: "high"},
        },
    }

    payload := []byte(`{"command": "get_status"}`)
    token := client.PublishWithProperties(ctx, "commands/device-001", payload, mqtt.QoS1, false, props)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }

    log.Println("Request envoyée, attente de la réponse sur responses/my-request")

    // Attendre la réponse...
    time.Sleep(10 * time.Second)
}
```

## Publisher avec retain

```go
package main

import (
    "context"
    "log"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("retain-publisher"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Message retenu - les nouveaux abonnés le recevront immédiatement
    token := client.Publish(ctx, "status/device-001", []byte("online"), mqtt.QoS1, true)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }
    log.Println("Status retenu publié")

    // Pour supprimer un message retenu, publier un payload vide avec retain=true
    // client.Publish(ctx, "status/device-001", []byte{}, mqtt.QoS1, true)
}
```

## Publisher QoS 2

```go
package main

import (
    "context"
    "log"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("qos2-publisher"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // QoS 2 - Exactly once delivery
    // Utilisé pour les transactions critiques
    payload := []byte(`{"transaction_id": "tx-12345", "amount": 100.00}`)

    token := client.Publish(ctx, "transactions/payments", payload, mqtt.QoS2, false)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }

    log.Println("Transaction publiée avec QoS 2 (exactly once)")
}
```

## Publisher haute performance avec pool

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    pool := mqtt.NewPool(
        mqtt.WithPoolSize(10),
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

    // Publier 10000 messages en parallèle
    var wg sync.WaitGroup
    start := time.Now()

    for i := 0; i < 10000; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()

            topic := fmt.Sprintf("test/msg/%d", n%100)
            payload := []byte(fmt.Sprintf("Message %d", n))

            if err := pool.Publish(ctx, topic, payload, mqtt.QoS0, false); err != nil {
                log.Printf("Erreur: %v", err)
            }
        }(i)
    }

    wg.Wait()
    elapsed := time.Since(start)

    log.Printf("10000 messages publiés en %v (%.0f msg/s)",
        elapsed, 10000/elapsed.Seconds())
}
```
