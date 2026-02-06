---
slug: /
---

# MQTT Client Library

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](./changelog)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](https://github.com/edgeo-scada/mqtt/blob/main/LICENSE)

Pure Go MQTT 5.0 client library with TLS, WebSocket, and connection pooling support.

## Overview

This library provides a complete implementation of the MQTT 5.0 protocol for Go, designed for industrial and IoT applications requiring reliable and high-performance communication.

## Features

### MQTT 5.0 Protocol
- Full MQTT 5.0 property support
- Detailed reason codes for diagnostics
- Topic aliases to reduce bandwidth
- Session expiry and message expiry
- Native Request/Response pattern

### Transports
- **TCP** - Standard connection on port 1883
- **TLS/SSL** - Secure connection on port 8883
- **WebSocket** - For environments with HTTP proxies
- **WebSocket Secure** - WebSocket over TLS

### Quality of Service
- **QoS 0** - At most once (fire and forget)
- **QoS 1** - At least once (acknowledged delivery)
- **QoS 2** - Exactly once (assured delivery)

### Advanced Features
- Automatic reconnection with exponential backoff
- Connection pooling for high performance
- Built-in metrics
- Topic wildcards (`+` and `#`)
- Will messages (Last Will and Testament)

## Installation

```bash
go get github.com/edgeo-scada/mqtt
```

## Quick Example

```go
package main

import (
    "context"
    "log"

    "github.com/edgeo-scada/mqtt/mqtt"
)

func main() {
    // Create the client
    client := mqtt.NewClient(
        mqtt.WithServer("mqtt://localhost:1883"),
        mqtt.WithClientID("my-app"),
    )

    // Connect
    if err := client.Connect(context.Background()); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(context.Background())

    // Publish a message
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
| [Getting Started](getting-started.md) | Quick start guide |
| [Client](client.md) | MQTT client API |
| [Options](options.md) | Configuration and options |
| [Pool](pool.md) | Connection pooling |
| [Errors](errors.md) | Error handling |
| [Metrics](metrics.md) | Metrics and monitoring |
| [CLI](cli.md) | edgeo-mqtt command-line tool |
| [Changelog](changelog.md) | Version history |

## Examples

| Example | Description |
|---------|-------------|
| [Publisher](examples/publisher.md) | Message publishing |
| [Subscriber](examples/subscriber.md) | Topic subscription |

## Architecture

```
mqtt/
├── client.go      # Main MQTT client
├── types.go       # Types and constants
├── protocol.go    # Protocol encoding/decoding
├── packets.go     # MQTT packets
├── options.go     # Configuration options
├── pool.go        # Connection pooling
├── metrics.go     # Metrics
├── errors.go      # Error handling
└── version.go     # Version information
```

## Compatibility

- Go 1.22+
- MQTT 5.0
- Tested brokers: Mosquitto, HiveMQ, EMQX, VerneMQ

## License

Apache License 2.0 - see [LICENSE](https://github.com/edgeo-scada/mqtt/blob/main/LICENSE) for details.
