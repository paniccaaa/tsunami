/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"sync"
	"time"
)

const (
	MaxLatencySamples       = 100000
	MaxLatencyHistorySamples = 1000 // For latency over time chart
)

// ErrorType represents the type of error that occurred
type ErrorType string

const (
	ErrorTypeTimeout          ErrorType = "timeout"
	ErrorTypeConnectionRefused ErrorType = "connection_refused"
	ErrorTypeDNS              ErrorType = "dns"
	ErrorTypeTLS              ErrorType = "tls"
	ErrorTypeOther            ErrorType = "other"
)

// GlobalMetrics holds aggregated metrics for a load test
type GlobalMetrics struct {
	TotalRequests uint64
	Successes     uint64
	Failures      uint64
	TotalLatency  time.Duration

	// Min/Max latency tracking
	MinLatency time.Duration
	MaxLatency time.Duration

	// Bytes transferred
	BytesSent     uint64
	BytesReceived uint64

	// Error breakdown
	ErrorTypes map[ErrorType]uint64

	// Latency history for over-time chart (sampled)
	LatencyHistory    []LatencyPoint
	latencyHistoryIdx int

	// Target RPS for comparison
	TargetRPS int

	// Test timing
	StartTime time.Time
	Duration  time.Duration // Configured duration (0 = infinite)

	Latencies   []time.Duration
	latencyIdx  int
	StatusCodes map[int]uint64

	sync.Mutex
}

// LatencyPoint represents a latency sample at a point in time
type LatencyPoint struct {
	Timestamp time.Time
	Latency   time.Duration
}

// NewGlobalMetrics creates a new GlobalMetrics instance
func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		Latencies:      make([]time.Duration, 0, MaxLatencySamples),
		StatusCodes:    make(map[int]uint64),
		ErrorTypes:     make(map[ErrorType]uint64),
		LatencyHistory: make([]LatencyPoint, 0, MaxLatencyHistorySamples),
		MinLatency:     time.Duration(1<<63 - 1), // Max duration as initial min
		MaxLatency:     0,
	}
}

// SetTestConfig sets the test configuration for progress tracking
func (m *GlobalMetrics) SetTestConfig(targetRPS int, duration time.Duration, startTime time.Time) {
	m.Lock()
	defer m.Unlock()
	m.TargetRPS = targetRPS
	m.Duration = duration
	m.StartTime = startTime
}

// AddLatency adds a latency sample to the metrics
func (m *GlobalMetrics) AddLatency(latency time.Duration) {
	// Update min/max
	if latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}

	// Add to latencies for percentile calculation
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

	// Add to latency history (sample every ~10 requests to avoid too much data)
	if m.TotalRequests%10 == 0 {
		point := LatencyPoint{
			Timestamp: time.Now(),
			Latency:   latency,
		}
		if len(m.LatencyHistory) < MaxLatencyHistorySamples {
			m.LatencyHistory = append(m.LatencyHistory, point)
		} else {
			m.LatencyHistory[m.latencyHistoryIdx] = point
			m.latencyHistoryIdx = (m.latencyHistoryIdx + 1) % MaxLatencyHistorySamples
		}
	}
}

// AddError records an error with its type
func (m *GlobalMetrics) AddError(errType ErrorType) {
	m.ErrorTypes[errType]++
}

// AddBytes adds bytes transferred
func (m *GlobalMetrics) AddBytes(sent, received uint64) {
	m.BytesSent += sent
	m.BytesReceived += received
}
