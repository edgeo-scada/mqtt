package mqtt

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Pool manages a pool of MQTT client connections.
type Pool struct {
	opts       *PoolOptions
	clients    []*poolClient
	clientOpts []Option
	mu         sync.RWMutex
	robin      uint64
	done       chan struct{}
	wg         sync.WaitGroup
	metrics    *PoolMetrics
}

type poolClient struct {
	client     *Client
	lastUsed   time.Time
	inFlight   int64
	mu         sync.Mutex
}

// PoolMetrics contains pool-specific metrics.
type PoolMetrics struct {
	TotalClients   Gauge
	HealthyClients Gauge
	TotalRequests  Counter
	FailedRequests Counter
}

// NewPool creates a new connection pool.
func NewPool(opts ...PoolOption) *Pool {
	options := NewPoolOptions()
	for _, opt := range opts {
		opt(options)
	}

	p := &Pool{
		opts:       options,
		clients:    make([]*poolClient, options.Size),
		clientOpts: options.ClientOptions,
		done:       make(chan struct{}),
		metrics: &PoolMetrics{},
	}

	return p
}

// Connect initializes all connections in the pool.
func (p *Pool) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	successCount := 0

	for i := 0; i < p.opts.Size; i++ {
		client := NewClient(p.clientOpts...)
		if err := client.Connect(ctx); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		p.clients[i] = &poolClient{
			client:   client,
			lastUsed: time.Now(),
		}
		successCount++
	}

	p.metrics.TotalClients.Set(int64(successCount))
	p.metrics.HealthyClients.Set(int64(successCount))

	if successCount == 0 {
		return firstErr
	}

	// Start health checker
	p.wg.Add(1)
	go p.healthChecker()

	return nil
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	close(p.done)
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, pc := range p.clients {
		if pc != nil && pc.client != nil {
			if err := pc.client.Disconnect(context.Background()); err != nil {
				lastErr = err
			}
		}
	}

	p.clients = nil
	p.metrics.TotalClients.Set(0)
	p.metrics.HealthyClients.Set(0)

	return lastErr
}

// Get returns a client from the pool using round-robin selection.
func (p *Pool) Get() (*Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.clients) == 0 {
		return nil, errors.New("mqtt: pool is empty")
	}

	p.metrics.TotalRequests.Add(1)

	// Round-robin with fallback to first healthy
	start := atomic.AddUint64(&p.robin, 1) % uint64(len(p.clients))

	for i := 0; i < len(p.clients); i++ {
		idx := (int(start) + i) % len(p.clients)
		pc := p.clients[idx]
		if pc != nil && pc.client != nil && pc.client.IsConnected() {
			pc.mu.Lock()
			pc.lastUsed = time.Now()
			atomic.AddInt64(&pc.inFlight, 1)
			pc.mu.Unlock()
			return pc.client, nil
		}
	}

	p.metrics.FailedRequests.Add(1)
	return nil, errors.New("mqtt: no healthy connections available")
}

// Release returns a client to the pool (decrements in-flight counter).
func (p *Pool) Release(client *Client) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, pc := range p.clients {
		if pc != nil && pc.client == client {
			atomic.AddInt64(&pc.inFlight, -1)
			return
		}
	}
}

// Publish publishes a message using a pooled connection.
func (p *Pool) Publish(ctx context.Context, topic string, payload []byte, qos QoS, retain bool) error {
	client, err := p.Get()
	if err != nil {
		return err
	}
	defer p.Release(client)

	token := client.Publish(ctx, topic, payload, qos, retain)
	return token.Wait()
}

// Subscribe subscribes using a pooled connection.
func (p *Pool) Subscribe(ctx context.Context, topic string, qos QoS, handler MessageHandler) error {
	client, err := p.Get()
	if err != nil {
		return err
	}
	defer p.Release(client)

	token := client.Subscribe(ctx, topic, qos, handler)
	return token.Wait()
}

func (p *Pool) healthChecker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.opts.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.checkHealth()
		}
	}
}

func (p *Pool) checkHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	healthy := int64(0)
	for i, pc := range p.clients {
		if pc == nil || pc.client == nil {
			// Try to create a new connection
			client := NewClient(p.clientOpts...)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := client.Connect(ctx); err == nil {
				p.clients[i] = &poolClient{
					client:   client,
					lastUsed: time.Now(),
				}
				healthy++
			}
			cancel()
			continue
		}

		if !pc.client.IsConnected() {
			// Try to reconnect
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := pc.client.Connect(ctx); err != nil {
				// Replace with new client
				pc.client = nil
				client := NewClient(p.clientOpts...)
				if err := client.Connect(ctx); err == nil {
					p.clients[i] = &poolClient{
						client:   client,
						lastUsed: time.Now(),
					}
					healthy++
				}
			} else {
				healthy++
			}
			cancel()
			continue
		}

		// Check idle timeout
		pc.mu.Lock()
		idle := time.Since(pc.lastUsed)
		inFlight := atomic.LoadInt64(&pc.inFlight)
		pc.mu.Unlock()

		if idle > p.opts.MaxIdleTime && inFlight == 0 {
			// Close idle connection but keep slot for later
			pc.client.Disconnect(context.Background())
			continue
		}

		healthy++
	}

	p.metrics.HealthyClients.Set(healthy)
}

// Metrics returns the pool metrics.
func (p *Pool) Metrics() *PoolMetrics {
	return p.metrics
}

// Size returns the pool size.
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

// HealthyCount returns the number of healthy connections.
func (p *Pool) HealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, pc := range p.clients {
		if pc != nil && pc.client != nil && pc.client.IsConnected() {
			count++
		}
	}
	return count
}
