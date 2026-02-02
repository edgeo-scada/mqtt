// Package transport provides low-level TCP transport functionality.
package transport

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"
)

// Transport interface for MQTT packet transport.
type Transport interface {
	// Connect establishes a connection.
	Connect(ctx context.Context) error
	// Close closes the connection.
	Close() error
	// Read reads data from the connection.
	Read(p []byte) (int, error)
	// Write writes data to the connection.
	Write(p []byte) (int, error)
	// SetDeadline sets read and write deadlines.
	SetDeadline(t time.Time) error
	// SetReadDeadline sets the read deadline.
	SetReadDeadline(t time.Time) error
	// SetWriteDeadline sets the write deadline.
	SetWriteDeadline(t time.Time) error
	// LocalAddr returns the local address.
	LocalAddr() net.Addr
	// RemoteAddr returns the remote address.
	RemoteAddr() net.Addr
}

// TCPTransport implements Transport over TCP.
type TCPTransport struct {
	address   string
	tlsConfig *tls.Config
	timeout   time.Duration
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	mu        sync.Mutex
}

// TCPOption is a functional option for TCPTransport.
type TCPOption func(*TCPTransport)

// WithTLSConfig sets the TLS configuration.
func WithTLSConfig(config *tls.Config) TCPOption {
	return func(t *TCPTransport) {
		t.tlsConfig = config
	}
}

// WithTimeout sets the connection timeout.
func WithTimeout(d time.Duration) TCPOption {
	return func(t *TCPTransport) {
		t.timeout = d
	}
}

// NewTCPTransport creates a new TCP transport.
func NewTCPTransport(address string, opts ...TCPOption) *TCPTransport {
	t := &TCPTransport{
		address: address,
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Connect establishes a TCP connection.
func (t *TCPTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	dialer := &net.Dialer{Timeout: t.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", t.address)
	if err != nil {
		return err
	}

	if t.tlsConfig != nil {
		tlsConn := tls.Client(conn, t.tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return err
		}
		conn = tlsConn
	}

	t.conn = conn
	t.reader = bufio.NewReaderSize(conn, 8192)
	t.writer = bufio.NewWriterSize(conn, 8192)

	return nil
}

// Close closes the TCP connection.
func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		err := t.conn.Close()
		t.conn = nil
		t.reader = nil
		t.writer = nil
		return err
	}
	return nil
}

// Read reads data from the connection.
func (t *TCPTransport) Read(p []byte) (int, error) {
	t.mu.Lock()
	reader := t.reader
	t.mu.Unlock()

	if reader == nil {
		return 0, io.EOF
	}
	return reader.Read(p)
}

// Write writes data to the connection.
func (t *TCPTransport) Write(p []byte) (int, error) {
	t.mu.Lock()
	writer := t.writer
	t.mu.Unlock()

	if writer == nil {
		return 0, io.ErrClosedPipe
	}

	n, err := writer.Write(p)
	if err != nil {
		return n, err
	}

	return n, writer.Flush()
}

// SetDeadline sets both read and write deadlines.
func (t *TCPTransport) SetDeadline(deadline time.Time) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return io.ErrClosedPipe
	}
	return conn.SetDeadline(deadline)
}

// SetReadDeadline sets the read deadline.
func (t *TCPTransport) SetReadDeadline(deadline time.Time) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return io.ErrClosedPipe
	}
	return conn.SetReadDeadline(deadline)
}

// SetWriteDeadline sets the write deadline.
func (t *TCPTransport) SetWriteDeadline(deadline time.Time) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return io.ErrClosedPipe
	}
	return conn.SetWriteDeadline(deadline)
}

// LocalAddr returns the local address.
func (t *TCPTransport) LocalAddr() net.Addr {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return nil
	}
	return conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (t *TCPTransport) RemoteAddr() net.Addr {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return nil
	}
	return conn.RemoteAddr()
}

// IsConnected returns true if connected.
func (t *TCPTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}
