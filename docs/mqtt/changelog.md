# Changelog

Historique des versions de la bibliothèque MQTT.

## [1.0.0] - 2025-02-01

### Première version stable

Version initiale de la bibliothèque client MQTT 5.0 en Go.

### Fonctionnalités

#### Protocole MQTT 5.0
- Support complet du protocole MQTT 5.0
- Tous les types de paquets (CONNECT, PUBLISH, SUBSCRIBE, etc.)
- Propriétés MQTT 5.0 (User Properties, Message Expiry, etc.)
- Codes de raison détaillés
- Topic aliases

#### Transports
- TCP standard (port 1883)
- TLS/SSL (port 8883)
- WebSocket (port 80)
- WebSocket Secure (port 443)

#### Qualité de Service
- QoS 0 (At most once)
- QoS 1 (At least once)
- QoS 2 (Exactly once)
- Gestion complète des flux QoS 2

#### Connexion
- Reconnexion automatique avec backoff exponentiel
- Keep-alive avec PING/PONG
- Clean Start / Session persistante
- Will Message (Last Will and Testament)

#### Souscriptions
- Wildcards (+ et #)
- Subscription options (NoLocal, Retain handling)
- Handlers par topic

#### Authentification
- Username/Password
- Certificats client (mTLS)
- Support TLS configurable

#### Fonctionnalités avancées
- Connection pooling
- Métriques intégrées
- Logging structuré (slog)
- Pattern functional options

### API

```go
// Création du client
client := mqtt.NewClient(opts...)

// Connexion
client.Connect(ctx)
client.Disconnect(ctx)

// Publication
client.Publish(ctx, topic, payload, qos, retain)
client.PublishWithProperties(ctx, topic, payload, qos, retain, props)

// Souscription
client.Subscribe(ctx, topic, qos, handler)
client.SubscribeMultiple(ctx, subs, handler)
client.Unsubscribe(ctx, topics...)

// État
client.IsConnected()
client.State()
client.Metrics()
client.ServerProperties()
```

### Options disponibles

- `WithServer(uri)` - URI du broker
- `WithClientID(id)` - Identifiant client
- `WithCredentials(user, pass)` - Authentification
- `WithCleanStart(bool)` - Clean session
- `WithKeepAlive(duration)` - Intervalle keep-alive
- `WithConnectTimeout(duration)` - Timeout connexion
- `WithAutoReconnect(bool)` - Reconnexion auto
- `WithTLS(config)` - Configuration TLS
- `WithWebSocket(bool)` - Transport WebSocket
- `WithWill(topic, payload, qos, retain)` - Will message
- Et plus...

### Dépendances

- `github.com/gorilla/websocket` v1.5.3 - Support WebSocket

---

## Roadmap

### [1.1.0] - Planifié

- [ ] Enhanced Authentication (AUTH packet)
- [ ] Shared Subscriptions
- [ ] Flow Control amélioré
- [ ] Métriques OpenTelemetry

### [1.2.0] - Planifié

- [ ] Message persistence
- [ ] Offline queue
- [ ] Compression des messages
- [ ] Rate limiting

---

## Convention de versioning

Ce projet suit [Semantic Versioning](https://semver.org/):

- **MAJOR**: Changements incompatibles de l'API
- **MINOR**: Nouvelles fonctionnalités rétrocompatibles
- **PATCH**: Corrections de bugs rétrocompatibles
