/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"sync"
	"time"
)

const (
	MaxLatencySamples = 100000
)

// GlobalMetrics holds aggregated metrics for a load test
type GlobalMetrics struct {
	TotalRequests uint64
	Successes     uint64
	Failures      uint64
	TotalLatency  time.Duration

	Latencies   []time.Duration
	latencyIdx  int
	StatusCodes map[int]uint64

	sync.Mutex
}

// NewGlobalMetrics creates a new GlobalMetrics instance
func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		Latencies:   make([]time.Duration, 0, MaxLatencySamples),
		StatusCodes: make(map[int]uint64),
	}
}

// AddLatency adds a latency sample to the metrics
func (m *GlobalMetrics) AddLatency(latency time.Duration) {
	if len(m.Latencies) < MaxLatencySamples {
		m.Latencies = append(m.Latencies, latency)
	} else {

		if cap(m.Latencies) < MaxLatencySamples {
			newLatencies := make([]time.Duration, MaxLatencySamples)
			copy(newLatencies, m.Latencies)
			m.Latencies = newLatencies
		}
		m.Latencies[m.latencyIdx] = latency
		m.latencyIdx = (m.latencyIdx + 1) % MaxLatencySamples
	}
}
