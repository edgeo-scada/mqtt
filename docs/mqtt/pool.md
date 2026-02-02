# Connection Pooling

Documentation du pool de connexions MQTT pour applications haute performance.

## Vue d'ensemble

Le pool de connexions maintient plusieurs connexions MQTT actives et les distribue via round-robin. Cela permet:

- **Haute disponibilité** - Failover automatique si une connexion échoue
- **Haute performance** - Distribution de charge entre connexions
- **Résilience** - Health checks et reconnexion automatique

## Création du pool

### NewPool

Crée un nouveau pool de connexions.

```go
func NewPool(opts ...PoolOption) *Pool
```

**Exemple:**

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

## Options du pool

### WithPoolSize

Définit le nombre de connexions dans le pool.

```go
mqtt.WithPoolSize(size int) PoolOption
```

**Défaut:** 3

### WithPoolMaxIdleTime

Définit la durée maximum d'inactivité d'une connexion.

```go
mqtt.WithPoolMaxIdleTime(d time.Duration) PoolOption
```

**Défaut:** 5 minutes

### WithPoolHealthCheckInterval

Définit l'intervalle des health checks.

```go
mqtt.WithPoolHealthCheckInterval(d time.Duration) PoolOption
```

**Défaut:** 30 secondes

### WithPoolClientOptions

Définit les options pour chaque client du pool.

```go
mqtt.WithPoolClientOptions(opts ...Option) PoolOption
```

## Méthodes

### Connect

Initialise toutes les connexions du pool.

```go
func (p *Pool) Connect(ctx context.Context) error
```

**Retourne:** `nil` si au moins une connexion réussit, sinon la première erreur.

**Exemple:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := pool.Connect(ctx); err != nil {
    log.Fatalf("Échec de connexion du pool: %v", err)
}
```

### Close

Ferme toutes les connexions du pool.

```go
func (p *Pool) Close() error
```

### Get

Obtient un client du pool (round-robin).

```go
func (p *Pool) Get() (*Client, error)
```

**Retourne:**
- `*Client` - Client MQTT disponible
- `error` - Si aucune connexion n'est disponible

**Exemple:**

```go
client, err := pool.Get()
if err != nil {
    log.Printf("Aucune connexion disponible: %v", err)
    return
}
defer pool.Release(client)

// Utiliser le client...
```

### Release

Retourne un client au pool.

```go
func (p *Pool) Release(client *Client)
```

### Publish

Publie un message via une connexion du pool.

```go
func (p *Pool) Publish(ctx context.Context, topic string, payload []byte, qos QoS, retain bool) error
```

**Exemple:**

```go
if err := pool.Publish(ctx, "sensors/data", payload, mqtt.QoS1, false); err != nil {
    log.Printf("Échec de publication: %v", err)
}
```

### Subscribe

Souscrit via une connexion du pool.

```go
func (p *Pool) Subscribe(ctx context.Context, topic string, qos QoS, handler MessageHandler) error
```

## Métriques du pool

### Metrics

Retourne les métriques du pool.

```go
func (p *Pool) Metrics() *PoolMetrics
```

```go
type PoolMetrics struct {
    TotalClients   Gauge    // Nombre total de clients
    HealthyClients Gauge    // Nombre de clients sains
    TotalRequests  Counter  // Total des requêtes
    FailedRequests Counter  // Requêtes échouées
}
```

### Size

Retourne la taille du pool.

```go
func (p *Pool) Size() int
```

### HealthyCount

Retourne le nombre de connexions saines.

```go
func (p *Pool) HealthyCount() int
```

## Exemple complet

### Application haute performance

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "sync"
    "time"

    "github.com/edgeo/drivers/mqtt/mqtt"
)

type SensorReading struct {
    SensorID  string  `json:"sensor_id"`
    Value     float64 `json:"value"`
    Timestamp int64   `json:"timestamp"`
}

func main() {
    // Créer le pool
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

    // Connexion
    ctx := context.Background()
    if err := pool.Connect(ctx); err != nil {
        log.Fatalf("Échec de connexion: %v", err)
    }
    defer pool.Close()

    log.Printf("Pool connecté: %d/%d connexions saines",
        pool.HealthyCount(), pool.Size())

    // Simuler des capteurs envoyant des données
    var wg sync.WaitGroup
    done := make(chan struct{})

    // 100 capteurs simulés
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
                        log.Printf("Erreur publication %s: %v", reading.SensorID, err)
                    }
                }
            }
        }(i)
    }

    // Afficher les métriques périodiquement
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

    // Attendre le signal d'arrêt
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Arrêt...")
    close(done)
    wg.Wait()
}
```

### Load balancing avec handlers différents

```go
package main

import (
    "context"
    "log"
    "sync"

    "github.com/edgeo/drivers/mqtt/mqtt"
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

    // Créer des handlers spécialisés sur différentes connexions
    var wg sync.WaitGroup

    // Handler pour les données de température
    wg.Add(1)
    go func() {
        defer wg.Done()
        client, err := pool.Get()
        if err != nil {
            log.Printf("Erreur get: %v", err)
            return
        }
        // Note: Ne pas Release si on garde le client pour les subscriptions

        handler := func(c *mqtt.Client, msg *mqtt.Message) {
            log.Printf("Temperature: %s", string(msg.Payload))
        }
        client.Subscribe(ctx, "sensors/+/temperature", mqtt.QoS1, handler)
    }()

    // Handler pour les données d'humidité
    wg.Add(1)
    go func() {
        defer wg.Done()
        client, err := pool.Get()
        if err != nil {
            log.Printf("Erreur get: %v", err)
            return
        }

        handler := func(c *mqtt.Client, msg *mqtt.Message) {
            log.Printf("Humidity: %s", string(msg.Payload))
        }
        client.Subscribe(ctx, "sensors/+/humidity", mqtt.QoS1, handler)
    }()

    wg.Wait()
    select {} // Bloquer
}
```

## Bonnes pratiques

### 1. Dimensionnement du pool

```go
// Règle générale: 1 connexion pour 1000 messages/sec
// Ajuster selon la latence réseau et la QoS utilisée
mqtt.WithPoolSize(numWorkers / 100)
```

### 2. Gestion des erreurs

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
// Surveiller la santé du pool
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        healthy := pool.HealthyCount()
        total := pool.Size()

        if healthy < total/2 {
            log.Warnf("Pool dégradé: %d/%d connexions", healthy, total)
            // Alerter...
        }
    }
}()
```

### 4. Graceful shutdown

```go
func shutdown(pool *mqtt.Pool) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Attendre que les publications en cours se terminent
    // ...

    if err := pool.Close(); err != nil {
        log.Printf("Erreur fermeture pool: %v", err)
    }
}
```
