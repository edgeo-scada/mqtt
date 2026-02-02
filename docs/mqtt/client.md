# Client API

Documentation complète de l'API du client MQTT.

## Création du client

### NewClient

Crée une nouvelle instance du client MQTT.

```go
func NewClient(opts ...Option) *Client
```

**Exemple:**

```go
client := mqtt.NewClient(
    mqtt.WithServer("mqtt://localhost:1883"),
    mqtt.WithClientID("my-client"),
    mqtt.WithKeepAlive(60*time.Second),
)
```

## Méthodes de connexion

### Connect

Établit une connexion au broker MQTT.

```go
func (c *Client) Connect(ctx context.Context) error
```

**Paramètres:**
- `ctx` - Context pour annulation et timeout

**Retour:**
- `error` - Erreur de connexion ou nil

**Exemple:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := client.Connect(ctx); err != nil {
    log.Fatalf("Connexion échouée: %v", err)
}
```

### Disconnect

Ferme proprement la connexion au broker.

```go
func (c *Client) Disconnect(ctx context.Context) error
```

**Exemple:**

```go
if err := client.Disconnect(context.Background()); err != nil {
    log.Printf("Erreur lors de la déconnexion: %v", err)
}
```

## Publication

### Publish

Publie un message sur un topic.

```go
func (c *Client) Publish(ctx context.Context, topic string, payload []byte, qos QoS, retain bool) *PublishToken
```

**Paramètres:**
- `ctx` - Context pour annulation
- `topic` - Topic de destination
- `payload` - Contenu du message
- `qos` - Niveau de QoS (0, 1, ou 2)
- `retain` - Si true, le message est retenu par le broker

**Retour:**
- `*PublishToken` - Token pour suivre l'opération asynchrone

**Exemple:**

```go
token := client.Publish(ctx, "sensors/temp", []byte("22.5"), mqtt.QoS1, false)
if err := token.Wait(); err != nil {
    log.Printf("Erreur de publication: %v", err)
}
```

### PublishWithProperties

Publie un message avec des propriétés MQTT 5.0.

```go
func (c *Client) PublishWithProperties(ctx context.Context, topic string, payload []byte, qos QoS, retain bool, props *Properties) *PublishToken
```

**Exemple:**

```go
props := &mqtt.Properties{
    ContentType:     "application/json",
    MessageExpiry:   ptrUint32(3600), // Expire après 1 heure
    ResponseTopic:   "response/topic",
    CorrelationData: []byte("request-123"),
    UserProperties: []mqtt.UserProperty{
        {Key: "source", Value: "sensor-1"},
    },
}

token := client.PublishWithProperties(ctx, "requests/data", payload, mqtt.QoS1, false, props)
```

## Souscription

### Subscribe

Souscrit à un topic avec un handler.

```go
func (c *Client) Subscribe(ctx context.Context, topic string, qos QoS, handler MessageHandler) *SubscribeToken
```

**Paramètres:**
- `ctx` - Context pour annulation
- `topic` - Filtre de topic (peut contenir wildcards)
- `qos` - QoS maximum demandé
- `handler` - Fonction appelée pour chaque message

**Exemple:**

```go
handler := func(c *mqtt.Client, msg *mqtt.Message) {
    log.Printf("Reçu sur %s: %s", msg.Topic, string(msg.Payload))
}

token := client.Subscribe(ctx, "sensors/#", mqtt.QoS1, handler)
if err := token.Wait(); err != nil {
    log.Printf("Erreur de souscription: %v", err)
}

// Vérifier le QoS accordé
log.Printf("QoS accordé: %v", token.GrantedQoS)
```

### SubscribeMultiple

Souscrit à plusieurs topics en une seule requête.

```go
func (c *Client) SubscribeMultiple(ctx context.Context, subs []Subscription, handler MessageHandler) *SubscribeToken
```

**Exemple:**

```go
subs := []mqtt.Subscription{
    {Topic: "sensors/temperature", QoS: mqtt.QoS1},
    {Topic: "sensors/humidity", QoS: mqtt.QoS1},
    {Topic: "alerts/#", QoS: mqtt.QoS2, NoLocal: true},
}

token := client.SubscribeMultiple(ctx, subs, handler)
if err := token.Wait(); err != nil {
    log.Printf("Erreur: %v", err)
}
```

### Unsubscribe

Se désabonne d'un ou plusieurs topics.

```go
func (c *Client) Unsubscribe(ctx context.Context, topics ...string) *UnsubscribeToken
```

**Exemple:**

```go
token := client.Unsubscribe(ctx, "sensors/temperature", "sensors/humidity")
if err := token.Wait(); err != nil {
    log.Printf("Erreur: %v", err)
}
```

## État de la connexion

### IsConnected

Vérifie si le client est connecté.

```go
func (c *Client) IsConnected() bool
```

### State

Retourne l'état actuel de la connexion.

```go
func (c *Client) State() ConnectionState
```

**États possibles:**
- `StateDisconnected` - Non connecté
- `StateConnecting` - Connexion en cours
- `StateConnected` - Connecté
- `StateDisconnecting` - Déconnexion en cours

**Exemple:**

```go
switch client.State() {
case mqtt.StateConnected:
    log.Println("Client connecté")
case mqtt.StateConnecting:
    log.Println("Connexion en cours...")
case mqtt.StateDisconnected:
    log.Println("Client déconnecté")
}
```

## Propriétés serveur

### ServerProperties

Retourne les propriétés envoyées par le serveur dans CONNACK.

```go
func (c *Client) ServerProperties() *Properties
```

**Exemple:**

```go
props := client.ServerProperties()
if props != nil {
    if props.MaximumQoS != nil {
        log.Printf("QoS maximum supporté: %d", *props.MaximumQoS)
    }
    if props.TopicAliasMaximum != nil {
        log.Printf("Topic alias max: %d", *props.TopicAliasMaximum)
    }
    if props.RetainAvailable != nil && *props.RetainAvailable == 0 {
        log.Println("Retain non supporté par le broker")
    }
}
```

## Métriques

### Metrics

Retourne les métriques du client.

```go
func (c *Client) Metrics() *Metrics
```

**Exemple:**

```go
metrics := client.Metrics()
snapshot := metrics.Snapshot()

log.Printf("Messages envoyés: %d", snapshot.MessagesSent)
log.Printf("Messages reçus: %d", snapshot.MessagesReceived)
log.Printf("Reconnexions: %d", snapshot.ReconnectAttempts)
```

## Tokens

Les opérations asynchrones retournent des tokens pour suivre leur état.

### Token interface

```go
type Token interface {
    Wait() error
    WaitTimeout(timeout time.Duration) error
    Done() <-chan struct{}
    Error() error
}
```

### Attente avec timeout

```go
token := client.Publish(ctx, topic, payload, mqtt.QoS1, false)
if err := token.WaitTimeout(5 * time.Second); err != nil {
    if err == mqtt.ErrTimeout {
        log.Println("Publication timeout")
    } else {
        log.Printf("Erreur: %v", err)
    }
}
```

### Utilisation avec select

```go
token := client.Subscribe(ctx, topic, mqtt.QoS1, handler)

select {
case <-token.Done():
    if err := token.Error(); err != nil {
        log.Printf("Erreur: %v", err)
    } else {
        log.Println("Souscription réussie")
    }
case <-time.After(10 * time.Second):
    log.Println("Timeout")
case <-ctx.Done():
    log.Println("Annulé")
}
```

## Types

### Message

Structure représentant un message reçu.

```go
type Message struct {
    Topic      string       // Topic du message
    Payload    []byte       // Contenu du message
    QoS        QoS          // Niveau de QoS
    Retain     bool         // Flag retain
    PacketID   uint16       // ID du paquet (QoS > 0)
    Properties *Properties  // Propriétés MQTT 5.0
    Duplicate  bool         // Flag de duplication
}
```

### MessageHandler

Signature du handler de messages.

```go
type MessageHandler func(client *Client, msg *Message)
```

### Subscription

Options de souscription.

```go
type Subscription struct {
    Topic             string         // Filtre de topic
    QoS               QoS            // QoS demandé
    NoLocal           bool           // Ne pas recevoir ses propres messages
    RetainAsPublished bool           // Garder le flag retain original
    RetainHandling    RetainHandling // Gestion des messages retenus
}
```

## Exemple complet

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

type SensorData struct {
    DeviceID    string  `json:"device_id"`
    Temperature float64 `json:"temperature"`
    Humidity    float64 `json:"humidity"`
    Timestamp   int64   `json:"timestamp"`
}

func main() {
    // Configuration du client
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("sensor-gateway"),
        mqtt.WithCleanStart(false),
        mqtt.WithSessionExpiryInterval(3600),
        mqtt.WithKeepAlive(30*time.Second),
        mqtt.WithAutoReconnect(true),
        mqtt.WithWill("gateways/sensor-gateway/status", []byte("offline"), mqtt.QoS1, true),
        mqtt.WithOnConnect(func(c *mqtt.Client) {
            log.Println("Connecté au broker")
            // Publier le statut online
            c.Publish(context.Background(), "gateways/sensor-gateway/status", []byte("online"), mqtt.QoS1, true)
        }),
        mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
            log.Printf("Connexion perdue: %v", err)
        }),
    )

    // Connexion
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Handler pour les commandes
    commandHandler := func(c *mqtt.Client, msg *mqtt.Message) {
        log.Printf("Commande reçue: %s", string(msg.Payload))

        // Répondre si un topic de réponse est spécifié
        if msg.Properties != nil && msg.Properties.ResponseTopic != "" {
            response := []byte(`{"status":"ok"}`)
            c.PublishWithProperties(ctx, msg.Properties.ResponseTopic, response, mqtt.QoS1, false,
                &mqtt.Properties{
                    CorrelationData: msg.Properties.CorrelationData,
                })
        }
    }

    // Souscrire aux commandes
    client.Subscribe(ctx, "gateways/sensor-gateway/commands", mqtt.QoS1, commandHandler)

    // Publier des données périodiquement
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)

    for {
        select {
        case <-sigCh:
            log.Println("Arrêt...")
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
                log.Printf("Erreur de publication: %v", err)
            }
        }
    }
}
```
