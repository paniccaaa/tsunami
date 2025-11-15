package cmd

import (
	"sync"
	"time"
)

const (
	maxLatencySamples = 100000
)

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

func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		Latencies:   make([]time.Duration, 0, maxLatencySamples),
		StatusCodes: make(map[int]uint64),
	}
}

func (m *GlobalMetrics) AddLatency(latency time.Duration) {
	if len(m.Latencies) < maxLatencySamples {
		m.Latencies = append(m.Latencies, latency)
	} else {

		if cap(m.Latencies) < maxLatencySamples {
			newLatencies := make([]time.Duration, maxLatencySamples)
			copy(newLatencies, m.Latencies)
			m.Latencies = newLatencies
		}
		m.Latencies[m.latencyIdx] = latency
		m.latencyIdx = (m.latencyIdx + 1) % maxLatencySamples
	}
}
