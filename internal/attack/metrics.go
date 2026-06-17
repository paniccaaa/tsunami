package attack

import (
	"sync"
	"time"
)

const (
	MaxLatencySamples        = 100000
	MaxLatencyHistorySamples = 1000
)

type ErrorType string

const (
	ErrorTypeTimeout          ErrorType = "timeout"
	ErrorTypeConnectionRefused ErrorType = "connection_refused"
	ErrorTypeDNS              ErrorType = "dns"
	ErrorTypeTLS              ErrorType = "tls"
	ErrorTypeOther            ErrorType = "other"
)

type GlobalMetrics struct {
	TotalRequests uint64
	Successes     uint64
	Failures      uint64
	TotalLatency  time.Duration

	MinLatency time.Duration
	MaxLatency time.Duration

	BytesSent     uint64
	BytesReceived uint64

	ErrorTypes map[ErrorType]uint64

	LatencyHistory    []LatencyPoint
	latencyHistoryIdx int

	TargetRPS int

	StartTime time.Time
	Duration  time.Duration

	Latencies   []time.Duration
	latencyIdx  int
	StatusCodes map[int]uint64

	sync.Mutex
}

type LatencyPoint struct {
	Timestamp time.Time
	Latency   time.Duration
}

func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		Latencies:      make([]time.Duration, 0, MaxLatencySamples),
		StatusCodes:    make(map[int]uint64),
		ErrorTypes:     make(map[ErrorType]uint64),
		LatencyHistory: make([]LatencyPoint, 0, MaxLatencyHistorySamples),
		MinLatency:     time.Duration(1<<63 - 1),
		MaxLatency:     0,
	}
}

func (m *GlobalMetrics) SetTestConfig(targetRPS int, duration time.Duration, startTime time.Time) {
	m.Lock()
	defer m.Unlock()
	m.TargetRPS = targetRPS
	m.Duration = duration
	m.StartTime = startTime
}

func (m *GlobalMetrics) AddLatency(latency time.Duration) {
	if latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}

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

func (m *GlobalMetrics) AddError(errType ErrorType) {
	m.ErrorTypes[errType]++
}

func (m *GlobalMetrics) AddBytes(sent, received uint64) {
	m.BytesSent += sent
	m.BytesReceived += received
}
