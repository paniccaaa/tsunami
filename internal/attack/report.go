/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"encoding/json"
	"maps"
	"time"
)

// Summary represents the test summary
type Summary struct {
	TotalRequests      uint64  `json:"total_requests"`
	SuccessfulRequests uint64  `json:"successful_requests"`
	FailedRequests     uint64  `json:"failed_requests"`
	TotalElapsedTime   string  `json:"total_elapsed_time"`
	AverageLatency     string  `json:"average_latency"`
	MinLatency         string  `json:"min_latency"`
	MaxLatency         string  `json:"max_latency"`
	ThroughputRPS      float64 `json:"throughput_rps"`
	TargetRPS          int     `json:"target_rps"`
	BytesSent          uint64  `json:"bytes_sent"`
	BytesReceived      uint64  `json:"bytes_received"`
}

// LatencyPercentiles represents latency percentiles
type LatencyPercentiles struct {
	P50 string `json:"p50"`
	P90 string `json:"p90"`
	P95 string `json:"p95"`
	P99 string `json:"p99"`
}

// AttackConfigJSON is a JSON-serializable version of AttackConfig
// that converts time.Duration fields to strings
type AttackConfigJSON struct {
	URL         string   `json:"url"`
	Method      string   `json:"method"`
	Body        string   `json:"body,omitempty"`
	Headers     []string `json:"headers,omitempty"`
	Duration    string   `json:"duration"`
	Timeout     string   `json:"timeout"`
	Workers     uint     `json:"workers"`
	Connections uint     `json:"connections"`
	RPS         int      `json:"rps"`
}

// ToJSON converts AttackConfig to AttackConfigJSON
func (cfg *AttackConfig) ToJSON() AttackConfigJSON {
	result := AttackConfigJSON{
		URL:         cfg.URL,
		Method:      cfg.Method,
		Workers:     cfg.Workers,
		Connections: cfg.Connections,
		RPS:         cfg.RPS,
		Duration:    cfg.Duration.String(),
		Timeout:     cfg.Timeout.String(),
	}
	if cfg.Body != "" {
		result.Body = cfg.Body
	}
	if len(cfg.Headers) > 0 {
		result.Headers = cfg.Headers
	}
	return result
}

// ErrorBreakdown represents error type breakdown
type ErrorBreakdown struct {
	Timeout          uint64 `json:"timeout"`
	ConnectionRefused uint64 `json:"connection_refused"`
	DNS              uint64 `json:"dns"`
	TLS              uint64 `json:"tls"`
	Other            uint64 `json:"other"`
}

// MetricsReport represents the full metrics report
type MetricsReport struct {
	Config             AttackConfigJSON   `json:"config"`
	Summary            Summary            `json:"summary"`
	LatencyPercentiles LatencyPercentiles `json:"latency_percentiles"`
	StatusCodes        map[int]uint64     `json:"status_codes"`
	ErrorBreakdown     ErrorBreakdown     `json:"error_breakdown"`
	Timestamp          string             `json:"timestamp"`
}

// ToJSON converts GlobalMetrics to JSON
func (m *GlobalMetrics) ToJSON(cfg *AttackConfig, elapsedTime time.Duration, reqPerSec float64, calculatePercentile func([]time.Duration, int) time.Duration) ([]byte, error) {
	m.Lock()
	defer m.Unlock()

	report := MetricsReport{}

	// Convert AttackConfig to JSON-serializable format
	report.Config = cfg.ToJSON()

	report.Summary.TotalRequests = m.TotalRequests
	report.Summary.SuccessfulRequests = m.Successes
	report.Summary.FailedRequests = m.Failures
	report.Summary.TotalElapsedTime = elapsedTime.Round(time.Second).String()

	if m.TotalRequests > 0 {
		avgLatency := m.TotalLatency / time.Duration(m.TotalRequests)
		report.Summary.AverageLatency = avgLatency.Round(time.Millisecond).String()
	}

	// Min/Max latency
	if m.MinLatency < time.Duration(1<<63-1) {
		report.Summary.MinLatency = m.MinLatency.Round(time.Millisecond).String()
	} else {
		report.Summary.MinLatency = "0ms"
	}
	report.Summary.MaxLatency = m.MaxLatency.Round(time.Millisecond).String()

	report.Summary.ThroughputRPS = reqPerSec
	report.Summary.TargetRPS = cfg.RPS

	// Bytes transferred
	report.Summary.BytesSent = m.BytesSent
	report.Summary.BytesReceived = m.BytesReceived

	if len(m.Latencies) > 0 {
		report.LatencyPercentiles.P50 = calculatePercentile(m.Latencies, 50).Round(time.Millisecond).String()
		report.LatencyPercentiles.P90 = calculatePercentile(m.Latencies, 90).Round(time.Millisecond).String()
		report.LatencyPercentiles.P95 = calculatePercentile(m.Latencies, 95).Round(time.Millisecond).String()
		report.LatencyPercentiles.P99 = calculatePercentile(m.Latencies, 99).Round(time.Millisecond).String()
	}

	report.StatusCodes = make(map[int]uint64)
	maps.Copy(report.StatusCodes, m.StatusCodes)

	// Error breakdown
	report.ErrorBreakdown = ErrorBreakdown{
		Timeout:          m.ErrorTypes[ErrorTypeTimeout],
		ConnectionRefused: m.ErrorTypes[ErrorTypeConnectionRefused],
		DNS:              m.ErrorTypes[ErrorTypeDNS],
		TLS:              m.ErrorTypes[ErrorTypeTLS],
		Other:            m.ErrorTypes[ErrorTypeOther],
	}

	report.Timestamp = time.Now().Format(time.RFC3339)

	return json.MarshalIndent(report, "", "  ")
}
