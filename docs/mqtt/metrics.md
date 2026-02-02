# Métriques et Monitoring

Documentation des métriques intégrées pour le monitoring du client MQTT.

## Vue d'ensemble

La bibliothèque fournit des métriques intégrées pour surveiller:
- Connexions et reconnexions
- Messages envoyés et reçus
- Latence des opérations
- Erreurs

## Accès aux métriques

### Métriques du client

```go
metrics := client.Metrics()
```

### Snapshot des métriques

```go
snapshot := metrics.Snapshot()
```

## Structure des métriques

### Metrics

```go
type Metrics struct {
    // Connexions
    ConnectionAttempts Counter  // Tentatives de connexion
    ActiveConnections  Gauge    // Connexions actives
    ReconnectAttempts  Counter  // Tentatives de reconnexion

    // Paquets
    PacketsSent     Counter     // Paquets envoyés
    PacketsReceived Counter     // Paquets reçus

    // Messages
    MessagesSent     Counter    // Messages publiés
    MessagesReceived Counter    // Messages reçus

    // Erreurs
    Errors Counter              // Total des erreurs

    // Latence
    PublishLatency   *LatencyHistogram  // Latence de publication
    SubscribeLatency *LatencyHistogram  // Latence de souscription
    ConnectLatency   *LatencyHistogram  // Latence de connexion

    // Temps
    StartTime time.Time         // Heure de démarrage
}
```

### MetricsSnapshot

```go
type MetricsSnapshot struct {
    ConnectionAttempts int64
    ActiveConnections  int64
    ReconnectAttempts  int64
    PacketsSent        int64
    PacketsReceived    int64
    MessagesSent       int64
    MessagesReceived   int64
    Errors             int64
    PublishLatency     LatencyStats
    SubscribeLatency   LatencyStats
    ConnectLatency     LatencyStats
    Uptime             time.Duration
}
```

## Types de métriques

### Counter

Compteur monotone croissant.

```go
type Counter struct {
    value int64
}

func (c *Counter) Add(delta int64)  // Incrémenter
func (c *Counter) Value() int64     // Valeur actuelle
func (c *Counter) Reset()           // Réinitialiser
```

### Gauge

Valeur pouvant augmenter ou diminuer.

```go
type Gauge struct {
    value int64
}

func (g *Gauge) Set(value int64)    // Définir la valeur
func (g *Gauge) Add(delta int64)    // Ajouter/soustraire
func (g *Gauge) Value() int64       // Valeur actuelle
```

### LatencyHistogram

Histogramme de latence avec statistiques.

```go
type LatencyHistogram struct {
    // ...
}

func (h *LatencyHistogram) Observe(latencyMs int64)
func (h *LatencyHistogram) ObserveDuration(d time.Duration)
func (h *LatencyHistogram) Stats() LatencyStats
```

### LatencyStats

Statistiques de latence.

```go
type LatencyStats struct {
    Count int64   // Nombre d'observations
    Sum   int64   // Somme des latences (ms)
    Min   int64   // Latence minimale (ms)
    Max   int64   // Latence maximale (ms)
    Avg   float64 // Latence moyenne (ms)
}
```

## Exemples d'utilisation

### Monitoring basique

```go
func printMetrics(client *mqtt.Client) {
    snapshot := client.Metrics().Snapshot()

    fmt.Printf("=== MQTT Metrics ===\n")
    fmt.Printf("Uptime: %v\n", snapshot.Uptime)
    fmt.Printf("Active connections: %d\n", snapshot.ActiveConnections)
    fmt.Printf("Connection attempts: %d\n", snapshot.ConnectionAttempts)
    fmt.Printf("Reconnect attempts: %d\n", snapshot.ReconnectAttempts)
    fmt.Printf("Messages sent: %d\n", snapshot.MessagesSent)
    fmt.Printf("Messages received: %d\n", snapshot.MessagesReceived)
    fmt.Printf("Packets sent: %d\n", snapshot.PacketsSent)
    fmt.Printf("Packets received: %d\n", snapshot.PacketsReceived)
    fmt.Printf("Errors: %d\n", snapshot.Errors)

    if snapshot.PublishLatency.Count > 0 {
        fmt.Printf("Publish latency: avg=%.2fms min=%dms max=%dms\n",
            snapshot.PublishLatency.Avg,
            snapshot.PublishLatency.Min,
            snapshot.PublishLatency.Max)
    }
}
```

### Monitoring périodique

```go
func startMetricsReporter(client *mqtt.Client, interval time.Duration) {
    ticker := time.NewTicker(interval)

    var lastSent, lastRecv int64

    go func() {
        for range ticker.C {
            snapshot := client.Metrics().Snapshot()

            // Calculer le débit
            sentRate := snapshot.MessagesSent - lastSent
            recvRate := snapshot.MessagesReceived - lastRecv
            lastSent = snapshot.MessagesSent
            lastRecv = snapshot.MessagesReceived

            log.Printf("MQTT stats: sent=%d/s recv=%d/s connected=%d errors=%d",
                sentRate/int64(interval.Seconds()),
                recvRate/int64(interval.Seconds()),
                snapshot.ActiveConnections,
                snapshot.Errors)
        }
    }()
}
```

### Export Prometheus

```go
import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type MQTTCollector struct {
    client *mqtt.Client

    connectionAttempts *prometheus.Desc
    activeConnections  *prometheus.Desc
    messagesSent       *prometheus.Desc
    messagesReceived   *prometheus.Desc
    publishLatency     *prometheus.Desc
}

func NewMQTTCollector(client *mqtt.Client) *MQTTCollector {
    return &MQTTCollector{
        client: client,
        connectionAttempts: prometheus.NewDesc(
            "mqtt_connection_attempts_total",
            "Total connection attempts",
            nil, nil,
        ),
        activeConnections: prometheus.NewDesc(
            "mqtt_active_connections",
            "Number of active connections",
            nil, nil,
        ),
        messagesSent: prometheus.NewDesc(
            "mqtt_messages_sent_total",
            "Total messages sent",
            nil, nil,
        ),
        messagesReceived: prometheus.NewDesc(
            "mqtt_messages_received_total",
            "Total messages received",
            nil, nil,
        ),
        publishLatency: prometheus.NewDesc(
            "mqtt_publish_latency_milliseconds",
            "Publish latency histogram",
            nil, nil,
        ),
    }
}

func (c *MQTTCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.connectionAttempts
    ch <- c.activeConnections
    ch <- c.messagesSent
    ch <- c.messagesReceived
    ch <- c.publishLatency
}

func (c *MQTTCollector) Collect(ch chan<- prometheus.Metric) {
    snapshot := c.client.Metrics().Snapshot()

    ch <- prometheus.MustNewConstMetric(
        c.connectionAttempts,
        prometheus.CounterValue,
        float64(snapshot.ConnectionAttempts),
    )
    ch <- prometheus.MustNewConstMetric(
        c.activeConnections,
        prometheus.GaugeValue,
        float64(snapshot.ActiveConnections),
    )
    ch <- prometheus.MustNewConstMetric(
        c.messagesSent,
        prometheus.CounterValue,
        float64(snapshot.MessagesSent),
    )
    ch <- prometheus.MustNewConstMetric(
        c.messagesReceived,
        prometheus.CounterValue,
        float64(snapshot.MessagesReceived),
    )
    ch <- prometheus.MustNewConstMetric(
        c.publishLatency,
        prometheus.GaugeValue,
        snapshot.PublishLatency.Avg,
    )
}

func main() {
    client := mqtt.NewClient(/* ... */)

    // Enregistrer le collector
    collector := NewMQTTCollector(client)
    prometheus.MustRegister(collector)

    // Endpoint HTTP pour Prometheus
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(":9090", nil)

    // ...
}
```

### Export JSON

```go
import "encoding/json"

func metricsHandler(client *mqtt.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        snapshot := client.Metrics().Snapshot()

        response := map[string]interface{}{
            "uptime_seconds":       snapshot.Uptime.Seconds(),
            "active_connections":   snapshot.ActiveConnections,
            "connection_attempts":  snapshot.ConnectionAttempts,
            "reconnect_attempts":   snapshot.ReconnectAttempts,
            "messages_sent":        snapshot.MessagesSent,
            "messages_received":    snapshot.MessagesReceived,
            "packets_sent":         snapshot.PacketsSent,
            "packets_received":     snapshot.PacketsReceived,
            "errors":               snapshot.Errors,
            "publish_latency": map[string]interface{}{
                "count": snapshot.PublishLatency.Count,
                "avg":   snapshot.PublishLatency.Avg,
                "min":   snapshot.PublishLatency.Min,
                "max":   snapshot.PublishLatency.Max,
            },
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
}
```

### Alerting

```go
func startAlertChecker(client *mqtt.Client) {
    ticker := time.NewTicker(time.Minute)

    var lastErrors int64

    go func() {
        for range ticker.C {
            snapshot := client.Metrics().Snapshot()

            // Alerte si trop d'erreurs
            errorRate := snapshot.Errors - lastErrors
            lastErrors = snapshot.Errors

            if errorRate > 10 {
                sendAlert(fmt.Sprintf("MQTT error rate high: %d errors/min", errorRate))
            }

            // Alerte si déconnecté
            if snapshot.ActiveConnections == 0 {
                sendAlert("MQTT client disconnected")
            }

            // Alerte si latence élevée
            if snapshot.PublishLatency.Avg > 1000 {
                sendAlert(fmt.Sprintf("MQTT publish latency high: %.0fms",
                    snapshot.PublishLatency.Avg))
            }
        }
    }()
}

func sendAlert(message string) {
    log.Printf("ALERT: %s", message)
    // Envoyer email, Slack, PagerDuty, etc.
}
```

## Métriques du pool

```go
poolMetrics := pool.Metrics()

fmt.Printf("Total clients: %d\n", poolMetrics.TotalClients.Value())
fmt.Printf("Healthy clients: %d\n", poolMetrics.HealthyClients.Value())
fmt.Printf("Total requests: %d\n", poolMetrics.TotalRequests.Value())
fmt.Printf("Failed requests: %d\n", poolMetrics.FailedRequests.Value())
```

## Réinitialisation

```go
// Réinitialiser toutes les métriques
client.Metrics().Reset()
```
