# MQTT Client Library

Bibliothèque client MQTT 5.0 en Go pur avec support TLS, WebSocket et connection pooling.

## Vue d'ensemble

Cette bibliothèque fournit une implémentation complète du protocole MQTT 5.0 pour Go, conçue pour les applications industrielles et IoT nécessitant une communication fiable et performante.

## Fonctionnalités

### Protocole MQTT 5.0
- Support complet des propriétés MQTT 5.0
- Codes de raison détaillés pour le diagnostic
- Topic aliases pour réduire la bande passante
- Session expiry et message expiry
- Request/Response pattern natif

### Transports
- **TCP** - Connexion standard sur port 1883
- **TLS/SSL** - Connexion sécurisée sur port 8883
- **WebSocket** - Pour environnements avec proxy HTTP
- **WebSocket Secure** - WebSocket sur TLS

### Qualité de Service
- **QoS 0** - At most once (fire and forget)
- **QoS 1** - At least once (acknowledged delivery)
- **QoS 2** - Exactly once (assured delivery)

### Fonctionnalités avancées
- Reconnexion automatique avec backoff exponentiel
- Connection pooling pour haute performance
- Métriques intégrées
- Wildcards topics (`+` et `#`)
- Will messages (Last Will and Testament)

## Installation

```bash
go get github.com/edgeo/drivers/mqtt
```

## Exemple rapide

```go
package main

import (
    "context"
    "log"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

func main() {
    // Créer le client
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-app"),
    )

    // Connexion
    if err := client.Connect(context.Background()); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(context.Background())

    // Publier un message
    token := client.Publish(context.Background(),
        "sensors/temperature",
        []byte("22.5"),
        mqtt.QoS1,
        false,
    )
    if err := token.Wait(); err != nil {
        log.Fatal(err)
    }
}
```

## Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](getting-started.md) | Guide de démarrage rapide |
| [Client](client.md) | API du client MQTT |
| [Options](options.md) | Configuration et options |
| [Pool](pool.md) | Connection pooling |
| [Errors](errors.md) | Gestion des erreurs |
| [Metrics](metrics.md) | Métriques et monitoring |
| [CLI](cli.md) | Outil en ligne de commande mqttcli |
| [Changelog](changelog.md) | Historique des versions |

## Exemples

| Exemple | Description |
|---------|-------------|
| [Publisher](examples/publisher.md) | Publication de messages |
| [Subscriber](examples/subscriber.md) | Souscription aux topics |

## Architecture

```
mqtt/
├── client.go      # Client MQTT principal
├── types.go       # Types et constantes
├── protocol.go    # Encodage/décodage protocole
├── packets.go     # Packets MQTT
├── options.go     # Options de configuration
├── pool.go        # Connection pooling
├── metrics.go     # Métriques
├── errors.go      # Gestion des erreurs
└── version.go     # Information de version
```

## Compatibilité

- Go 1.22+
- MQTT 5.0
- Brokers testés : Mosquitto, HiveMQ, EMQX, VerneMQ

## Licence

MIT License
