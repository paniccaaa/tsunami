package cmd

import (
	"sync"
	"time"
)

const (
	// maxLatencySamples limits the number of latency samples stored in memory
	// to prevent excessive memory usage during long-running tests.
	// 100,000 samples provide accurate percentile calculations (~800KB memory).
	maxLatencySamples = 100000
)

type GlobalMetrics struct {
	TotalRequests uint64
	Successes     uint64
	Failures      uint64
	TotalLatency  time.Duration

	// Latencies is a circular buffer that stores at most maxLatencySamples values
	Latencies   []time.Duration
	latencyIdx  int // current position in circular buffer
	StatusCodes map[int]uint64

	sync.Mutex
}

func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		Latencies:   make([]time.Duration, 0, maxLatencySamples),
		StatusCodes: make(map[int]uint64),
	}
}

// AddLatency adds a latency sample to the circular buffer.
// If the buffer is full, it overwrites the oldest value.
func (m *GlobalMetrics) AddLatency(latency time.Duration) {
	if len(m.Latencies) < maxLatencySamples {
		// Buffer not full yet, append
		m.Latencies = append(m.Latencies, latency)
	} else {
		// Buffer is full, use circular buffer behavior
		m.Latencies[m.latencyIdx] = latency
		m.latencyIdx = (m.latencyIdx + 1) % maxLatencySamples
	}
}
