package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/edgeo/drivers/mqtt/mqtt"
)

// createClient creates and connects an MQTT client with the current configuration.
func createClient(ctx context.Context) (*mqtt.Client, error) {
	opts := buildClientOptions()
	client := mqtt.NewClient(opts...)

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Connect(connectCtx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	return client, nil
}

// buildClientOptions builds MQTT client options from the current configuration.
func buildClientOptions() []mqtt.Option {
	opts := []mqtt.Option{
		mqtt.WithServer(broker),
		mqtt.WithKeepAlive(keepAlive),
		mqtt.WithConnectTimeout(timeout),
		mqtt.WithAutoReconnect(false),
	}

	if clientID != "" {
		opts = append(opts, mqtt.WithClientID(clientID))
	} else {
		opts = append(opts, mqtt.WithClientID(fmt.Sprintf("edgeo-mqtt-%d", time.Now().UnixNano())))
	}

	if username != "" {
		opts = append(opts, mqtt.WithCredentials(username, password))
	}

	// TLS configuration
	tlsConfig, err := buildTLSConfig()
	if err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Warning: TLS config error: %v\n", err)
	}
	if tlsConfig != nil {
		opts = append(opts, mqtt.WithTLS(tlsConfig))
	}

	if verbose {
		opts = append(opts, mqtt.WithOnConnect(func(c *mqtt.Client) {
			fmt.Fprintln(os.Stderr, "Connected to broker")
		}))
		opts = append(opts, mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
			fmt.Fprintf(os.Stderr, "Connection lost: %v\n", err)
		}))
		// Enable debug logging
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		opts = append(opts, mqtt.WithLogger(logger))
	}

	return opts
}

// buildTLSConfig builds TLS configuration from flags.
func buildTLSConfig() (*tls.Config, error) {
	if caFile == "" && certFile == "" && keyFile == "" && !insecure {
		return nil, nil
	}

	config := &tls.Config{
		InsecureSkipVerify: insecure,
	}

	// Load CA certificate
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		config.RootCAs = caCertPool
	}

	// Load client certificate and key
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	return config, nil
}

// disconnectClient gracefully disconnects the client.
func disconnectClient(client *mqtt.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client.Disconnect(ctx)
}

// printVerbose prints a message if verbose mode is enabled.
func printVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// printError prints an error message to stderr.
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// parseQoS parses a QoS level from an integer.
func parseQoS(qos int) (mqtt.QoS, error) {
	switch qos {
	case 0:
		return mqtt.QoS0, nil
	case 1:
		return mqtt.QoS1, nil
	case 2:
		return mqtt.QoS2, nil
	default:
		return 0, fmt.Errorf("invalid QoS level: %d (must be 0, 1, or 2)", qos)
	}
}

// parseUserProperties parses user properties from key=value strings.
func parseUserProperties(props []string) []mqtt.UserProperty {
	var userProps []mqtt.UserProperty
	for _, p := range props {
		for i := 0; i < len(p); i++ {
			if p[i] == '=' {
				userProps = append(userProps, mqtt.UserProperty{
					Key:   p[:i],
					Value: p[i+1:],
				})
				break
			}
		}
	}
	return userProps
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// formatBytes formats bytes for display.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
