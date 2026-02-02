# edgeo-mqtt - MQTT Command Line Interface

A complete MQTT 5.0 command-line client for testing, debugging, monitoring, and benchmarking MQTT brokers.

## Installation

```bash
go build -o edgeo-mqtt ./cmd/edgeo-mqtt
```

Or with version information:

```bash
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o edgeo-mqtt ./cmd/edgeo-mqtt
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `pub` | Publish messages to a topic |
| `sub` | Subscribe to topics and receive messages |
| `watch` | Monitor topics in real-time with live updates |
| `info` | Show broker connection information |
| `topics` | Discover available topics |
| `interactive` | Interactive REPL shell mode |
| `bench` | Performance benchmarking tools |
| `version` | Print version information |

## Global Flags

These flags are available for all commands:

### Connection

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--broker` | `-b` | `mqtt://localhost:1883` | MQTT broker URI |
| `--client-id` | | auto-generated | Client identifier |
| `--username` | `-u` | | Username for authentication |
| `--password` | `-P` | | Password for authentication |
| `--timeout` | | `30s` | Connection timeout |
| `--keepalive` | | `60s` | Keep-alive interval |

### TLS

| Flag | Description |
|------|-------------|
| `--ca-file` | CA certificate file |
| `--cert-file` | Client certificate file |
| `--key-file` | Client private key file |
| `--insecure` | Skip TLS certificate verification |

### Output

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Output format: table, json, csv, raw |
| `--verbose` | `-v` | `false` | Verbose output |
| `--no-color` | | `false` | Disable colored output |

### Configuration

| Flag | Description |
|------|-------------|
| `--config` | Config file path (default: `~/.edgeo-mqtt.yaml`) |

## Broker URI Formats

Supported URI schemes:

- `mqtt://host:port` - Plain TCP (default port: 1883)
- `mqtts://host:port` - TLS encrypted (default port: 8883)
- `ws://host:port/path` - WebSocket (default port: 80)
- `wss://host:port/path` - Secure WebSocket (default port: 443)

## Command: pub

Publish messages to an MQTT topic.

### Usage

```bash
edgeo-mqtt pub -t <topic> -m <message> [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Topic to publish to (required) |
| `--message` | `-m` | Message payload |
| `--file` | `-f` | Read payload from file |
| `--qos` | `-q` | QoS level: 0, 1, or 2 (default: 0) |
| `--retain` | `-r` | Retain message |
| `--count` | `-n` | Number of messages (default: 1, 0=infinite) |
| `--interval` | `-i` | Interval between messages |

### MQTT 5.0 Properties

| Flag | Description |
|------|-------------|
| `--content-type` | Content type (MIME type) |
| `--response-topic` | Response topic for request/response |
| `--correlation-data` | Correlation data |
| `--user-prop` | User property (key=value, repeatable) |
| `--payload-format` | Payload format: 0=bytes, 1=UTF-8 |
| `--message-expiry` | Message expiry interval in seconds |

### Examples

```bash
# Simple publish
edgeo-mqtt pub -t "sensor/temp" -m "23.5"

# QoS 1 with retain
edgeo-mqtt pub -t "config/device" -m '{"enabled":true}' -q 1 --retain

# From file
edgeo-mqtt pub -t "data/batch" -f payload.json

# From stdin
echo "hello" | edgeo-mqtt pub -t "test"
cat data.json | edgeo-mqtt pub -t "data"

# Repeated publishing
edgeo-mqtt pub -t "heartbeat" -m "ping" -n 100 -i 1s

# Template with counter ({n} is replaced)
edgeo-mqtt pub -t "test" -m "message-{n}" -n 10

# MQTT 5.0 request/response
edgeo-mqtt pub -t "request/123" -m '{"action":"get"}' \
  --response-topic "response/123" \
  --correlation-data "req-001" \
  --content-type "application/json"
```

## Command: sub

Subscribe to topics and display received messages.

### Usage

```bash
edgeo-mqtt sub -t <topic> [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Topics to subscribe (repeatable, required) |
| `--qos` | `-q` | QoS level: 0, 1, or 2 (default: 0) |
| `--count` | `-c` | Max messages to receive (0=unlimited) |
| `--timestamps` | | Show message timestamps |
| `--show-properties` | | Show MQTT 5.0 properties |
| `--no-local` | | Don't receive own messages (MQTT 5.0) |
| `--retain-handling` | | Retain handling: 0=send, 1=new only, 2=none |

### Examples

```bash
# Subscribe to single topic
edgeo-mqtt sub -t "sensor/temp"

# Wildcards
edgeo-mqtt sub -t "sensor/#"
edgeo-mqtt sub -t "sensor/+/temp"

# Multiple topics
edgeo-mqtt sub -t "sensor/#" -t "actuator/#"

# Limit messages
edgeo-mqtt sub -t "events" -c 10

# JSON output with timestamps
edgeo-mqtt sub -t "data" -o json --timestamps

# Show MQTT 5.0 properties
edgeo-mqtt sub -t "request/#" --show-properties
```

## Command: watch

Monitor topics in real-time with a continuously updating display.

### Usage

```bash
edgeo-mqtt watch -t <topic> [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Topics to watch (repeatable, required) |
| `--diff` | | Only show changed values |
| `--alert-high` | | Alert when numeric value exceeds threshold |
| `--alert-low` | | Alert when numeric value below threshold |
| `--log` | | Log messages to CSV file |
| `--refresh` | | Screen refresh interval (default: 500ms) |

### Examples

```bash
# Watch with live updates
edgeo-mqtt watch -t "sensor/#"

# Show only changes
edgeo-mqtt watch -t "sensor/temp" --diff

# Threshold alerts
edgeo-mqtt watch -t "sensor/temp" --alert-high 30 --alert-low 10

# Log to file
edgeo-mqtt watch -t "data/#" --log messages.csv
```

## Command: info

Display broker connection information and server properties.

### Usage

```bash
edgeo-mqtt info [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--sys` | Read $SYS topics for broker statistics |

### Examples

```bash
# Basic connection info
edgeo-mqtt info

# Include broker statistics
edgeo-mqtt info --sys

# JSON output
edgeo-mqtt info -o json
```

## Command: topics

Discover available topics by listening for messages.

### Usage

```bash
edgeo-mqtt topics [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Topic pattern to subscribe (default: "#") |
| `--duration` | `-d` | Discovery duration (default: 5s) |
| `--sys` | | Discover $SYS topics |

### Examples

```bash
# Discover all topics (5 seconds)
edgeo-mqtt topics -d 5s

# Discover specific pattern
edgeo-mqtt topics -t "sensor/#" -d 30s

# List $SYS broker topics
edgeo-mqtt topics --sys
```

## Command: interactive

Start an interactive MQTT shell session.

### Usage

```bash
edgeo-mqtt interactive [flags]
```

### Shell Commands

| Command | Description |
|---------|-------------|
| `connect [broker]` | Connect to broker |
| `disconnect` | Disconnect from broker |
| `status` | Show connection status |
| `pub <topic> <msg> [qos] [retain]` | Publish message |
| `sub <topic> [qos]` | Subscribe to topic |
| `unsub <topic>` | Unsubscribe from topic |
| `topics` | List active subscriptions |
| `clear` | Clear screen |
| `help` | Show help |
| `quit` / `exit` | Exit shell |

### Example Session

```
$ edgeo-mqtt interactive -b mqtt://localhost:1883
MQTT Interactive Shell
Type 'help' for available commands

mqtt[connected@localhost]> sub sensor/#
Subscribed to sensor/# with QoS 0

mqtt[connected@localhost]> pub sensor/temp 23.5
Published to sensor/temp (QoS 0, Retain: false)

mqtt[connected@localhost]> status

  Status:              Connected
  Broker:              mqtt://localhost:1883
  Subscriptions:       1
  Messages received:   0

mqtt[connected@localhost]> quit
Goodbye!
```

## Command: bench

Run performance benchmarks on MQTT operations.

### Subcommands

| Command | Description |
|---------|-------------|
| `bench pub` | Benchmark publishing |
| `bench sub` | Benchmark subscribing |
| `bench pubsub` | Benchmark round-trip latency |

### Common Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--topic` | `-t` | `edgeo-mqtt/bench` | Topic for benchmark |
| `--count` | `-n` | `10000` | Number of messages |
| `--clients` | `-c` | `1` | Number of concurrent clients |
| `--payload` | `-s` | `100` | Payload size in bytes |
| `--qos` | `-q` | `0` | QoS level |
| `--warmup` | | `100` | Warmup messages (not counted) |

### Examples

```bash
# Benchmark publishing (10000 messages, 10 clients)
edgeo-mqtt bench pub -t "bench" -n 10000 -c 10

# Benchmark with larger payload
edgeo-mqtt bench pub -t "bench" -n 5000 -s 1024

# Benchmark subscribing
edgeo-mqtt bench sub -t "bench/#" -c 5 -d 30s

# Measure round-trip latency
edgeo-mqtt bench pubsub -t "bench" -n 1000

# QoS 1 benchmark
edgeo-mqtt bench pub -t "bench" -n 1000 -q 1
```

### Output Metrics

- **Messages**: Total messages sent/received
- **Errors**: Failed operations
- **Duration**: Total benchmark time
- **Throughput**: Messages per second
- **Bandwidth**: Bytes per second
- **Latency**: Min, Avg, Max, P99, StdDev

## Configuration File

Create `~/.edgeo-mqtt.yaml` for default settings:

```yaml
# Connection
broker: mqtt://localhost:1883
client-id: edgeo-mqtt-default
username: user
password: secret

# Timeouts
timeout: 30s
keepalive: 60s

# TLS
ca-file: /path/to/ca.crt
cert-file: /path/to/client.crt
key-file: /path/to/client.key
insecure: false

# Output
output: table
verbose: false
no-color: false
```

Environment variables are also supported with `MQTTCLI_` prefix:

```bash
export MQTTCLI_BROKER=mqtt://broker:1883
export MQTTCLI_USERNAME=user
export MQTTCLI_PASSWORD=secret
```

## Output Formats

### Table (default)

```
sensor/temp [QoS0] 23.5
sensor/humidity [QoS0,R] 65
```

### JSON

```bash
edgeo-mqtt sub -t "sensor/#" -o json
```

```json
{"topic":"sensor/temp","payload":"23.5","qos":0,"retain":false}
```

### CSV

```bash
edgeo-mqtt sub -t "sensor/#" -o csv --timestamps
```

```csv
timestamp,topic,payload,qos,retain
2024-01-15T10:30:00Z,sensor/temp,23.5,0,false
```

### Raw

```bash
edgeo-mqtt sub -t "sensor/#" -o raw
```

```
23.5
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (connection failed, publish failed, etc.) |

## See Also

- [MQTT 5.0 Specification](https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html)
- [Client Library Documentation](client.md)
- [Examples](examples/)
