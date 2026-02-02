package mqtt

import (
	"crypto/tls"
	"log/slog"
	"net/url"
	"time"
)

// ClientOptions contains configuration options for the MQTT client.
type ClientOptions struct {
	// Servers is a list of broker URIs to connect to.
	Servers []*url.URL
	// ClientID is the client identifier.
	ClientID string
	// Username for authentication.
	Username string
	// Password for authentication.
	Password []byte
	// CleanStart indicates whether to start a new session.
	CleanStart bool
	// KeepAlive is the keep-alive interval.
	KeepAlive time.Duration
	// ConnectTimeout is the connection timeout.
	ConnectTimeout time.Duration
	// WriteTimeout is the write timeout.
	WriteTimeout time.Duration
	// MaxReconnectInterval is the maximum reconnection interval.
	MaxReconnectInterval time.Duration
	// AutoReconnect enables automatic reconnection.
	AutoReconnect bool
	// ConnectRetryInterval is the initial retry interval.
	ConnectRetryInterval time.Duration
	// MaxRetries is the maximum number of reconnection attempts (0 = unlimited).
	MaxRetries int
	// SessionExpiryInterval is the session expiry in seconds.
	SessionExpiryInterval uint32
	// ReceiveMaximum is the max number of QoS 1/2 messages to process concurrently.
	ReceiveMaximum uint16
	// MaxPacketSize is the maximum packet size the client will accept.
	MaxPacketSize uint32
	// TopicAliasMaximum is the max number of topic aliases.
	TopicAliasMaximum uint16
	// RequestResponseInfo requests response information from the server.
	RequestResponseInfo bool
	// RequestProblemInfo requests problem information from the server.
	RequestProblemInfo bool
	// UserProperties are user-defined properties for CONNECT.
	UserProperties []UserProperty
	// TLSConfig is the TLS configuration.
	TLSConfig *tls.Config
	// WebSocket enables WebSocket transport.
	WebSocket bool
	// WebSocketPath is the WebSocket endpoint path.
	WebSocketPath string
	// Will message configuration.
	WillEnabled    bool
	WillTopic      string
	WillPayload    []byte
	WillQoS        QoS
	WillRetain     bool
	WillProperties *Properties
	// Callbacks
	OnConnect         OnConnectHandler
	OnConnectionLost  ConnectionLostHandler
	OnReconnecting    ReconnectHandler
	DefaultMsgHandler MessageHandler
	// Logger
	Logger *slog.Logger
}

// NewClientOptions creates ClientOptions with default values.
func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		CleanStart:           true,
		KeepAlive:            DefaultKeepAlive,
		ConnectTimeout:       DefaultConnectTimeout,
		WriteTimeout:         DefaultWriteTimeout,
		AutoReconnect:        true,
		MaxReconnectInterval: DefaultMaxReconnectDelay,
		ConnectRetryInterval: time.Second,
		MaxRetries:           0,
		ReceiveMaximum:       DefaultReceiveMaximum,
		MaxPacketSize:        DefaultMaxPacketSize,
		TopicAliasMaximum:    DefaultTopicAliasMaximum,
		RequestProblemInfo:   true,
		WebSocketPath:        "/mqtt",
	}
}

// Option is a functional option for configuring the client.
type Option func(*ClientOptions)

// WithServer adds a server URI.
func WithServer(uri string) Option {
	return func(o *ClientOptions) {
		u, err := url.Parse(uri)
		if err == nil {
			o.Servers = append(o.Servers, u)
		}
	}
}

// WithServers sets multiple server URIs.
func WithServers(uris ...string) Option {
	return func(o *ClientOptions) {
		for _, uri := range uris {
			u, err := url.Parse(uri)
			if err == nil {
				o.Servers = append(o.Servers, u)
			}
		}
	}
}

// WithClientID sets the client identifier.
func WithClientID(id string) Option {
	return func(o *ClientOptions) {
		o.ClientID = id
	}
}

// WithCredentials sets username and password.
func WithCredentials(username, password string) Option {
	return func(o *ClientOptions) {
		o.Username = username
		o.Password = []byte(password)
	}
}

// WithCleanStart sets the clean start flag.
func WithCleanStart(clean bool) Option {
	return func(o *ClientOptions) {
		o.CleanStart = clean
	}
}

// WithKeepAlive sets the keep-alive interval.
func WithKeepAlive(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.KeepAlive = d
	}
}

// WithConnectTimeout sets the connection timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.ConnectTimeout = d
	}
}

// WithWriteTimeout sets the write timeout.
func WithWriteTimeout(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.WriteTimeout = d
	}
}

// WithAutoReconnect enables or disables automatic reconnection.
func WithAutoReconnect(enabled bool) Option {
	return func(o *ClientOptions) {
		o.AutoReconnect = enabled
	}
}

// WithMaxReconnectInterval sets the maximum reconnection interval.
func WithMaxReconnectInterval(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.MaxReconnectInterval = d
	}
}

// WithConnectRetryInterval sets the initial reconnection interval.
func WithConnectRetryInterval(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.ConnectRetryInterval = d
	}
}

// WithMaxRetries sets the maximum number of reconnection attempts.
func WithMaxRetries(n int) Option {
	return func(o *ClientOptions) {
		o.MaxRetries = n
	}
}

// WithSessionExpiryInterval sets the session expiry interval.
func WithSessionExpiryInterval(seconds uint32) Option {
	return func(o *ClientOptions) {
		o.SessionExpiryInterval = seconds
	}
}

// WithReceiveMaximum sets the receive maximum.
func WithReceiveMaximum(max uint16) Option {
	return func(o *ClientOptions) {
		o.ReceiveMaximum = max
	}
}

// WithMaxPacketSize sets the maximum packet size.
func WithMaxPacketSize(size uint32) Option {
	return func(o *ClientOptions) {
		o.MaxPacketSize = size
	}
}

// WithTopicAliasMaximum sets the topic alias maximum.
func WithTopicAliasMaximum(max uint16) Option {
	return func(o *ClientOptions) {
		o.TopicAliasMaximum = max
	}
}

// WithRequestResponseInfo sets the request response info flag.
func WithRequestResponseInfo(enabled bool) Option {
	return func(o *ClientOptions) {
		o.RequestResponseInfo = enabled
	}
}

// WithRequestProblemInfo sets the request problem info flag.
func WithRequestProblemInfo(enabled bool) Option {
	return func(o *ClientOptions) {
		o.RequestProblemInfo = enabled
	}
}

// WithUserProperties sets user properties for CONNECT.
func WithUserProperties(props []UserProperty) Option {
	return func(o *ClientOptions) {
		o.UserProperties = props
	}
}

// WithTLS sets the TLS configuration.
func WithTLS(config *tls.Config) Option {
	return func(o *ClientOptions) {
		o.TLSConfig = config
	}
}

// WithTLSInsecure enables TLS without certificate verification.
func WithTLSInsecure() Option {
	return func(o *ClientOptions) {
		o.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

// WithWebSocket enables WebSocket transport.
func WithWebSocket(enabled bool) Option {
	return func(o *ClientOptions) {
		o.WebSocket = enabled
	}
}

// WithWebSocketPath sets the WebSocket endpoint path.
func WithWebSocketPath(path string) Option {
	return func(o *ClientOptions) {
		o.WebSocketPath = path
	}
}

// WithWill sets the Will message.
func WithWill(topic string, payload []byte, qos QoS, retain bool) Option {
	return func(o *ClientOptions) {
		o.WillEnabled = true
		o.WillTopic = topic
		o.WillPayload = payload
		o.WillQoS = qos
		o.WillRetain = retain
	}
}

// WithWillProperties sets the Will message properties.
func WithWillProperties(props *Properties) Option {
	return func(o *ClientOptions) {
		o.WillProperties = props
	}
}

// WithOnConnect sets the connection callback.
func WithOnConnect(handler OnConnectHandler) Option {
	return func(o *ClientOptions) {
		o.OnConnect = handler
	}
}

// WithOnConnectionLost sets the connection lost callback.
func WithOnConnectionLost(handler ConnectionLostHandler) Option {
	return func(o *ClientOptions) {
		o.OnConnectionLost = handler
	}
}

// WithOnReconnecting sets the reconnecting callback.
func WithOnReconnecting(handler ReconnectHandler) Option {
	return func(o *ClientOptions) {
		o.OnReconnecting = handler
	}
}

// WithDefaultMessageHandler sets the default message handler.
func WithDefaultMessageHandler(handler MessageHandler) Option {
	return func(o *ClientOptions) {
		o.DefaultMsgHandler = handler
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *ClientOptions) {
		o.Logger = logger
	}
}

// PoolOptions contains configuration options for the connection pool.
type PoolOptions struct {
	// Size is the number of connections in the pool.
	Size int
	// MaxIdleTime is the maximum time a connection can be idle.
	MaxIdleTime time.Duration
	// HealthCheckInterval is the interval between health checks.
	HealthCheckInterval time.Duration
	// ClientOptions are the options for each client in the pool.
	ClientOptions []Option
}

// NewPoolOptions creates PoolOptions with default values.
func NewPoolOptions() *PoolOptions {
	return &PoolOptions{
		Size:                3,
		MaxIdleTime:         5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// PoolOption is a functional option for configuring the pool.
type PoolOption func(*PoolOptions)

// WithPoolSize sets the pool size.
func WithPoolSize(size int) PoolOption {
	return func(o *PoolOptions) {
		o.Size = size
	}
}

// WithPoolMaxIdleTime sets the maximum idle time.
func WithPoolMaxIdleTime(d time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.MaxIdleTime = d
	}
}

// WithPoolHealthCheckInterval sets the health check interval.
func WithPoolHealthCheckInterval(d time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.HealthCheckInterval = d
	}
}

// WithPoolClientOptions sets client options for pool connections.
func WithPoolClientOptions(opts ...Option) PoolOption {
	return func(o *PoolOptions) {
		o.ClientOptions = opts
	}
}
