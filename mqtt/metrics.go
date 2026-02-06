// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mqtt

import (
	"sync"
	"sync/atomic"
	"time"
)

// Counter is a simple atomic counter.
type Counter struct {
	value int64
}

// Add adds a value to the counter.
func (c *Counter) Add(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// Reset resets the counter to zero.
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// Gauge is a simple atomic gauge that can go up and down.
type Gauge struct {
	value int64
}

// Set sets the gauge value.
func (g *Gauge) Set(value int64) {
	atomic.StoreInt64(&g.value, value)
}

// Add adds a value to the gauge.
func (g *Gauge) Add(delta int64) {
	atomic.AddInt64(&g.value, delta)
}

// Value returns the current gauge value.
func (g *Gauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}

// LatencyHistogram tracks latency distribution.
type LatencyHistogram struct {
	mu       sync.RWMutex
	count    int64
	sum      int64
	min      int64
	max      int64
	buckets  []int64
	bounds   []int64
}

// NewLatencyHistogram creates a new latency histogram.
func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		min:     -1,
		bounds:  []int64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		buckets: make([]int64, 13), // 12 buckets + overflow
	}
}

// Observe records a latency observation in milliseconds.
func (h *LatencyHistogram) Observe(latencyMs int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.count++
	h.sum += latencyMs

	if h.min < 0 || latencyMs < h.min {
		h.min = latencyMs
	}
	if latencyMs > h.max {
		h.max = latencyMs
	}

	// Find bucket
	for i, bound := range h.bounds {
		if latencyMs <= bound {
			h.buckets[i]++
			return
		}
	}
	h.buckets[len(h.buckets)-1]++ // overflow
}

// ObserveDuration records a duration.
func (h *LatencyHistogram) ObserveDuration(d time.Duration) {
	h.Observe(d.Milliseconds())
}

// Stats returns histogram statistics.
func (h *LatencyHistogram) Stats() LatencyStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := LatencyStats{
		Count: h.count,
		Sum:   h.sum,
		Min:   h.min,
		Max:   h.max,
	}

	if h.count > 0 {
		stats.Avg = float64(h.sum) / float64(h.count)
	}

	return stats
}

// LatencyStats contains latency statistics.
type LatencyStats struct {
	Count int64
	Sum   int64
	Min   int64
	Max   int64
	Avg   float64
}

// Metrics contains all client metrics.
type Metrics struct {
	// Connection metrics
	ConnectionAttempts Counter
	ActiveConnections  Gauge
	ReconnectAttempts  Counter

	// Packet metrics
	PacketsSent     Counter
	PacketsReceived Counter

	// Message metrics
	MessagesSent     Counter
	MessagesReceived Counter

	// Error metrics
	Errors Counter

	// Latency metrics
	PublishLatency   *LatencyHistogram
	SubscribeLatency *LatencyHistogram
	ConnectLatency   *LatencyHistogram

	// Start time
	StartTime time.Time
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		PublishLatency:   NewLatencyHistogram(),
		SubscribeLatency: NewLatencyHistogram(),
		ConnectLatency:   NewLatencyHistogram(),
		StartTime:        time.Now(),
	}
}

// Snapshot returns a copy of the current metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		ConnectionAttempts: m.ConnectionAttempts.Value(),
		ActiveConnections:  m.ActiveConnections.Value(),
		ReconnectAttempts:  m.ReconnectAttempts.Value(),
		PacketsSent:        m.PacketsSent.Value(),
		PacketsReceived:    m.PacketsReceived.Value(),
		MessagesSent:       m.MessagesSent.Value(),
		MessagesReceived:   m.MessagesReceived.Value(),
		Errors:             m.Errors.Value(),
		PublishLatency:     m.PublishLatency.Stats(),
		SubscribeLatency:   m.SubscribeLatency.Stats(),
		ConnectLatency:     m.ConnectLatency.Stats(),
		Uptime:             time.Since(m.StartTime),
	}
}

// MetricsSnapshot is a point-in-time snapshot of metrics.
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

// Reset resets all metrics.
func (m *Metrics) Reset() {
	m.ConnectionAttempts.Reset()
	m.ActiveConnections.Set(0)
	m.ReconnectAttempts.Reset()
	m.PacketsSent.Reset()
	m.PacketsReceived.Reset()
	m.MessagesSent.Reset()
	m.MessagesReceived.Reset()
	m.Errors.Reset()
	m.PublishLatency = NewLatencyHistogram()
	m.SubscribeLatency = NewLatencyHistogram()
	m.ConnectLatency = NewLatencyHistogram()
	m.StartTime = time.Now()
}
