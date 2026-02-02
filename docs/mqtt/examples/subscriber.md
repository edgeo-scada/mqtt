# Exemple Subscriber

Exemples de souscription aux messages MQTT.

## Subscriber basique

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Reçu sur %s: %s", msg.Topic, string(msg.Payload))
    }

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("basic-subscriber"),
        mqtt.WithDefaultMessageHandler(handler),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    token := client.Subscribe(ctx, "test/topic", mqtt.QoS1, handler)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }
    log.Println("Souscrit à test/topic")

    // Attendre Ctrl+C
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Arrêt...")
}
```

## Subscriber avec wildcards

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("[%s] %s", msg.Topic, string(msg.Payload))
    }

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("wildcard-subscriber"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Single-level wildcard: sensors/+/temperature
    // Matche: sensors/room1/temperature, sensors/room2/temperature
    // Ne matche pas: sensors/building1/room1/temperature
    client.Subscribe(ctx, "sensors/+/temperature", mqtt.QoS1, handler)

    // Multi-level wildcard: sensors/#
    // Matche: sensors/anything, sensors/room1/temp, sensors/a/b/c/d
    client.Subscribe(ctx, "alerts/#", mqtt.QoS1, handler)

    log.Println("Souscrit aux topics avec wildcards")

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
}
```

## Subscriber multi-topics

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("multi-subscriber"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Handler pour chaque type de données
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Reçu: topic=%s payload=%s qos=%d retain=%v",
            msg.Topic, string(msg.Payload), msg.QoS, msg.Retain)
    }

    // Souscrire à plusieurs topics en une requête
    subs := []mqtt.Subscription{
        {Topic: "sensors/temperature", QoS: mqtt.QoS1},
        {Topic: "sensors/humidity", QoS: mqtt.QoS1},
        {Topic: "sensors/pressure", QoS: mqtt.QoS0},
        {Topic: "alerts/critical", QoS: mqtt.QoS2},
    }

    token := client.SubscribeMultiple(ctx, subs, handler)
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }

    // Vérifier les QoS accordés
    for i, qos := range token.GrantedQoS {
        log.Printf("Topic %s: QoS demandé=%d, accordé=%d",
            subs[i].Topic, subs[i].QoS, qos)
    }

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
}
```

## Subscriber JSON avec décodage

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

type SensorData struct {
    DeviceID    string  `json:"device_id"`
    Temperature float64 `json:"temperature"`
    Humidity    float64 `json:"humidity"`
    Timestamp   string  `json:"timestamp"`
}

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        var data SensorData
        if err := json.Unmarshal(msg.Payload, &data); err != nil {
            log.Printf("Erreur décodage JSON: %v", err)
            return
        }

        log.Printf("Capteur %s: temp=%.1f°C humidity=%.1f%% time=%s",
            data.DeviceID, data.Temperature, data.Humidity, data.Timestamp)

        // Alertes
        if data.Temperature > 30 {
            log.Printf("ALERTE: Température élevée sur %s!", data.DeviceID)
        }
    }

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("json-subscriber"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    client.Subscribe(ctx, "sensors/+/data", mqtt.QoS1, handler)
    log.Println("En attente de données capteurs...")

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
}
```

## Subscriber avec session persistante

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Reçu: %s -> %s", msg.Topic, string(msg.Payload))
    }

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("persistent-subscriber"), // ID fixe requis
        mqtt.WithCleanStart(false),                 // Reprendre la session
        mqtt.WithSessionExpiryInterval(86400),      // Session expire après 24h
        mqtt.WithOnConnect(func(c *mqtt.Client) {
            log.Println("Connecté")
            // Les souscriptions sont conservées dans la session
        }),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Souscrire seulement si nouvelle session
    // (les souscriptions sont conservées sinon)
    token := client.Subscribe(ctx, "important/messages", mqtt.QoS1, handler)
    token.Wait()

    log.Println("Session persistante active")
    log.Println("Les messages QoS1/2 seront conservés pendant votre absence")

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    // Déconnexion normale - la session est conservée
    // Les messages arrivant pendant la déconnexion seront reçus à la reconnexion
}
```

## Subscriber avec handlers différenciés

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("multi-handler-subscriber"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Handler pour les températures
    tempHandler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("🌡️  Temperature: %s", string(msg.Payload))
    }

    // Handler pour l'humidité
    humidityHandler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("💧 Humidity: %s", string(msg.Payload))
    }

    // Handler pour les alertes
    alertHandler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("🚨 ALERT: %s", string(msg.Payload))
        // Envoyer notification, email, etc.
    }

    // Handler pour les commandes
    commandHandler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("⚙️  Command received: %s", string(msg.Payload))

        var cmd struct {
            Action string `json:"action"`
        }
        json.Unmarshal(msg.Payload, &cmd)

        // Répondre si topic de réponse spécifié
        if msg.Properties != nil && msg.Properties.ResponseTopic != "" {
            response := []byte(`{"status":"ok"}`)
            c.PublishWithProperties(ctx, msg.Properties.ResponseTopic, response, mqtt.QoS1, false,
                &mqtt.Properties{CorrelationData: msg.Properties.CorrelationData})
        }
    }

    // Souscrire avec handlers spécifiques
    client.Subscribe(ctx, "sensors/+/temperature", mqtt.QoS1, tempHandler)
    client.Subscribe(ctx, "sensors/+/humidity", mqtt.QoS1, humidityHandler)
    client.Subscribe(ctx, "alerts/#", mqtt.QoS2, alertHandler)
    client.Subscribe(ctx, "commands/my-device", mqtt.QoS1, commandHandler)

    log.Println("En écoute sur plusieurs topics avec handlers différenciés...")

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
}
```

## Subscriber avec reconnexion et resouscription

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Message: %s -> %s", msg.Topic, string(msg.Payload))
    }

    topics := []string{"sensors/#", "commands/device-001", "status/+"}

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("resilient-subscriber"),
        mqtt.WithAutoReconnect(true),
        mqtt.WithConnectRetryInterval(time.Second),
        mqtt.WithMaxReconnectInterval(30*time.Second),
        mqtt.WithOnConnect(func(c *mqtt.Client) {
            log.Println("Connecté - resouscription aux topics...")

            for _, topic := range topics {
                token := c.Subscribe(context.Background(), topic, mqtt.QoS1, handler)
                if err := token.Wait(); err != nil {
                    log.Printf("Erreur souscription %s: %v", topic, err)
                } else {
                    log.Printf("Souscrit à %s", topic)
                }
            }
        }),
        mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
            log.Printf("Connexion perdue: %v - reconnexion automatique...", err)
        }),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
}
```

## Subscriber avec désabonnement

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Reçu: %s", string(msg.Payload))
    }

    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("unsub-demo"),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Souscrire
    client.Subscribe(ctx, "test/topic", mqtt.QoS1, handler)
    log.Println("Souscrit à test/topic")

    time.Sleep(10 * time.Second)

    // Se désabonner
    token := client.Unsubscribe(ctx, "test/topic")
    if err := token.Wait(); err != nil {
        log.Printf("Erreur désabonnement: %v", err)
    }
    log.Println("Désabonné de test/topic")

    time.Sleep(5 * time.Second)
}
```
