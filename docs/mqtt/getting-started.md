# Getting Started

Guide de démarrage rapide pour la bibliothèque MQTT.

## Prérequis

- Go 1.22 ou supérieur
- Un broker MQTT (Mosquitto, HiveMQ, EMQX, etc.)

## Installation

```bash
go get github.com/edgeo-scada/mqtt
```

## Premier client

### Publisher simple

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    // Créer le client avec les options de base
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-publisher"),
        mqtt.WithCleanStart(true),
    )

    // Connexion au broker
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Échec de connexion: %v", err)
    }
    defer client.Disconnect(context.Background())

    log.Println("Connecté au broker MQTT")

    // Publier un message
    topic := "test/hello"
    payload := []byte("Hello MQTT!")

    token := client.Publish(context.Background(), topic, payload, mqtt.QoS1, false)
    if err := token.Wait(); err != nil {
        log.Fatalf("Échec de publication: %v", err)
    }

    log.Printf("Message publié sur %s", topic)
}
```

### Subscriber simple

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
    // Handler pour les messages reçus
    handler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Message reçu sur %s: %s", msg.Topic, string(msg.Payload))
    }

    // Créer le client
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-subscriber"),
        mqtt.WithDefaultMessageHandler(handler),
    )

    // Connexion
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Échec de connexion: %v", err)
    }
    defer client.Disconnect(context.Background())

    // Souscrire au topic
    token := client.Subscribe(context.Background(), "test/#", mqtt.QoS1, handler)
    if err := token.Wait(); err != nil {
        log.Fatalf("Échec de souscription: %v", err)
    }
    log.Println("Souscrit à test/#")

    // Attendre le signal d'arrêt
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Arrêt...")
}
```

## Connexion sécurisée (TLS)

```go
import "crypto/tls"

client := mqtt.NewClient(
    mqtt.WithServer("mqtts://broker.example.com:8883"),
    mqtt.WithClientID("secure-client"),
    mqtt.WithTLS(&tls.Config{
        // Configuration TLS personnalisée si nécessaire
    }),
)
```

Pour les tests sans vérification de certificat :

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtts://localhost:8883"),
    mqtt.WithTLSInsecure(),
)
```

## Connexion WebSocket

```go
// WebSocket standard
client := mqtt.NewClient(
    mqtt.WithServer("ws://broker.example.com:80"),
    mqtt.WithWebSocketPath("/mqtt"),
)

// WebSocket sécurisé
client := mqtt.NewClient(
    mqtt.WithServer("wss://broker.example.com:443"),
    mqtt.WithWebSocketPath("/mqtt"),
)
```

## Authentification

### Username/Password

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithCredentials("username", "password"),
)
```

### Certificat client (mTLS)

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

## Niveaux de QoS

### QoS 0 - At most once

Message envoyé une seule fois, sans confirmation. Peut être perdu.

```go
client.Publish(ctx, "topic", payload, mqtt.QoS0, false)
```

### QoS 1 - At least once

Message confirmé par PUBACK. Peut être dupliqué.

```go
token := client.Publish(ctx, "topic", payload, mqtt.QoS1, false)
if err := token.Wait(); err != nil {
    log.Printf("Erreur: %v", err)
}
```

### QoS 2 - Exactly once

Livraison garantie exactement une fois (PUBREC/PUBREL/PUBCOMP).

```go
token := client.Publish(ctx, "topic", payload, mqtt.QoS2, false)
if err := token.Wait(); err != nil {
    log.Printf("Erreur: %v", err)
}
```

## Wildcards

### Single-level wildcard (+)

Correspond à un seul niveau de topic.

```go
// Souscrit à sensors/room1/temperature, sensors/room2/temperature, etc.
client.Subscribe(ctx, "sensors/+/temperature", mqtt.QoS1, handler)
```

### Multi-level wildcard (#)

Correspond à tous les niveaux restants.

```go
// Souscrit à tous les topics sous sensors/
client.Subscribe(ctx, "sensors/#", mqtt.QoS1, handler)
```

## Will Message (Last Will and Testament)

Configure un message envoyé automatiquement si le client se déconnecte de façon inattendue.

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

## Reconnexion automatique

Par défaut, le client se reconnecte automatiquement avec backoff exponentiel.

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithAutoReconnect(true),
    mqtt.WithConnectRetryInterval(1*time.Second),
    mqtt.WithMaxReconnectInterval(2*time.Minute),
    mqtt.WithMaxRetries(0), // 0 = illimité
    mqtt.WithOnReconnecting(func(c *mqtt.Client, opts *mqtt.ClientOptions) {
        log.Println("Tentative de reconnexion...")
    }),
)
```

## Prochaines étapes

- [Client API](client.md) - Documentation complète de l'API client
- [Options](options.md) - Toutes les options de configuration
- [Pool](pool.md) - Connection pooling pour haute performance
- [Metrics](metrics.md) - Monitoring et métriques
