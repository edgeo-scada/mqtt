# Options de configuration

Documentation complète des options de configuration du client MQTT.

## Options de connexion

### WithServer

Ajoute un URI de broker MQTT.

```go
mqtt.WithServer(uri string) Option
```

**Schémas supportés:**
- `mqtt://` ou `tcp://` - TCP standard (port 1883)
- `mqtts://`, `ssl://` ou `tls://` - TLS (port 8883)
- `ws://` - WebSocket (port 80)
- `wss://` - WebSocket Secure (port 443)

**Exemple:**

```go
mqtt.WithServer("mqtt://localhost:1883")
mqtt.WithServer("mqtts://broker.example.com:8883")
mqtt.WithServer("wss://broker.example.com/mqtt")
```

### WithServers

Configure plusieurs brokers pour le failover.

```go
mqtt.WithServers(uris ...string) Option
```

**Exemple:**

```go
mqtt.WithServers(
    "mqtt://broker1.example.com:1883",
    "mqtt://broker2.example.com:1883",
    "mqtt://broker3.example.com:1883",
)
```

### WithClientID

Définit l'identifiant du client.

```go
mqtt.WithClientID(id string) Option
```

**Exemple:**

```go
mqtt.WithClientID("my-application-001")
```

### WithCleanStart

Active ou désactive le clean start (nouvelle session).

```go
mqtt.WithCleanStart(clean bool) Option
```

- `true` - Démarre une nouvelle session (défaut)
- `false` - Reprend la session existante si disponible

### WithKeepAlive

Définit l'intervalle de keep-alive.

```go
mqtt.WithKeepAlive(d time.Duration) Option
```

**Défaut:** 60 secondes

**Exemple:**

```go
mqtt.WithKeepAlive(30 * time.Second)
```

### WithConnectTimeout

Définit le timeout de connexion.

```go
mqtt.WithConnectTimeout(d time.Duration) Option
```

**Défaut:** 30 secondes

### WithWriteTimeout

Définit le timeout d'écriture.

```go
mqtt.WithWriteTimeout(d time.Duration) Option
```

**Défaut:** 10 secondes

## Options d'authentification

### WithCredentials

Configure l'authentification par username/password.

```go
mqtt.WithCredentials(username, password string) Option
```

**Exemple:**

```go
mqtt.WithCredentials("user", "secret123")
```

## Options TLS

### WithTLS

Configure TLS avec une configuration personnalisée.

```go
mqtt.WithTLS(config *tls.Config) Option
```

**Exemple:**

```go
// Avec certificat client
cert, _ := tls.LoadX509KeyPair("client.crt", "client.key")

mqtt.WithTLS(&tls.Config{
    Certificates:       []tls.Certificate{cert},
    InsecureSkipVerify: false,
    MinVersion:         tls.VersionTLS12,
})
```

### WithTLSInsecure

Désactive la vérification du certificat serveur.

```go
mqtt.WithTLSInsecure() Option
```

> **Attention:** À utiliser uniquement pour les tests.

## Options WebSocket

### WithWebSocket

Active le transport WebSocket.

```go
mqtt.WithWebSocket(enabled bool) Option
```

### WithWebSocketPath

Définit le chemin de l'endpoint WebSocket.

```go
mqtt.WithWebSocketPath(path string) Option
```

**Défaut:** `/mqtt`

**Exemple:**

```go
mqtt.WithWebSocketPath("/ws/mqtt")
```

## Options de reconnexion

### WithAutoReconnect

Active ou désactive la reconnexion automatique.

```go
mqtt.WithAutoReconnect(enabled bool) Option
```

**Défaut:** true

### WithConnectRetryInterval

Définit l'intervalle initial de retry.

```go
mqtt.WithConnectRetryInterval(d time.Duration) Option
```

**Défaut:** 1 seconde

### WithMaxReconnectInterval

Définit l'intervalle maximum de retry (backoff).

```go
mqtt.WithMaxReconnectInterval(d time.Duration) Option
```

**Défaut:** 2 minutes

### WithMaxRetries

Définit le nombre maximum de tentatives de reconnexion.

```go
mqtt.WithMaxRetries(n int) Option
```

**Défaut:** 0 (illimité)

## Options MQTT 5.0

### WithSessionExpiryInterval

Définit la durée de vie de la session en secondes.

```go
mqtt.WithSessionExpiryInterval(seconds uint32) Option
```

- `0` - Session supprimée à la déconnexion
- `0xFFFFFFFF` - Session permanente

**Exemple:**

```go
mqtt.WithSessionExpiryInterval(3600) // 1 heure
```

### WithReceiveMaximum

Définit le nombre maximum de messages QoS 1/2 en vol.

```go
mqtt.WithReceiveMaximum(max uint16) Option
```

**Défaut:** 65535

### WithMaxPacketSize

Définit la taille maximum de paquet acceptée.

```go
mqtt.WithMaxPacketSize(size uint32) Option
```

**Défaut:** 256 MB

### WithTopicAliasMaximum

Définit le nombre maximum de topic aliases.

```go
mqtt.WithTopicAliasMaximum(max uint16) Option
```

**Défaut:** 0 (désactivé)

### WithRequestResponseInfo

Demande les informations de réponse au serveur.

```go
mqtt.WithRequestResponseInfo(enabled bool) Option
```

### WithRequestProblemInfo

Demande les informations de problème au serveur.

```go
mqtt.WithRequestProblemInfo(enabled bool) Option
```

**Défaut:** true

### WithUserProperties

Définit les propriétés utilisateur pour CONNECT.

```go
mqtt.WithUserProperties(props []UserProperty) Option
```

**Exemple:**

```go
mqtt.WithUserProperties([]mqtt.UserProperty{
    {Key: "app-version", Value: "1.0.0"},
    {Key: "client-type", Value: "sensor"},
})
```

## Options Will Message

### WithWill

Configure le message Last Will and Testament.

```go
mqtt.WithWill(topic string, payload []byte, qos QoS, retain bool) Option
```

**Exemple:**

```go
mqtt.WithWill(
    "devices/sensor-1/status",
    []byte("offline"),
    mqtt.QoS1,
    true,
)
```

### WithWillProperties

Configure les propriétés du Will message.

```go
mqtt.WithWillProperties(props *Properties) Option
```

**Exemple:**

```go
mqtt.WithWillProperties(&mqtt.Properties{
    WillDelayInterval: ptrUint32(60), // Délai de 60 secondes
    MessageExpiry:     ptrUint32(3600),
    ContentType:       "application/json",
})
```

## Callbacks

### WithOnConnect

Callback appelé après une connexion réussie.

```go
mqtt.WithOnConnect(handler OnConnectHandler) Option

type OnConnectHandler func(client *Client)
```

**Exemple:**

```go
mqtt.WithOnConnect(func(c *mqtt.Client) {
    log.Println("Connecté!")
    // Resouscrire aux topics
    c.Subscribe(context.Background(), "my/topic", mqtt.QoS1, handler)
})
```

### WithOnConnectionLost

Callback appelé quand la connexion est perdue.

```go
mqtt.WithOnConnectionLost(handler ConnectionLostHandler) Option

type ConnectionLostHandler func(client *Client, err error)
```

**Exemple:**

```go
mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
    log.Printf("Connexion perdue: %v", err)
})
```

### WithOnReconnecting

Callback appelé avant chaque tentative de reconnexion.

```go
mqtt.WithOnReconnecting(handler ReconnectHandler) Option

type ReconnectHandler func(client *Client, opts *ClientOptions)
```

**Exemple:**

```go
mqtt.WithOnReconnecting(func(c *mqtt.Client, opts *mqtt.ClientOptions) {
    log.Println("Tentative de reconnexion...")
})
```

### WithDefaultMessageHandler

Handler par défaut pour les messages sans handler spécifique.

```go
mqtt.WithDefaultMessageHandler(handler MessageHandler) Option

type MessageHandler func(client *Client, msg *Message)
```

## Options de logging

### WithLogger

Configure le logger.

```go
mqtt.WithLogger(logger *slog.Logger) Option
```

**Exemple:**

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

mqtt.WithLogger(logger)
```

## Exemple complet

```go
client := mqtt.NewClient(
    // Connexion
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

    // Reconnexion
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
        log.Println("Connecté")
    }),
    mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
        log.Printf("Déconnecté: %v", err)
    }),

    // Logging
    mqtt.WithLogger(slog.Default()),
)
```
