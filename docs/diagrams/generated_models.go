// Package models contains data model types and interfaces for the Tsunami
// HTTP load testing tool.
//
// This file is generated from the UML class diagram (docs/diagrams/classes.puml)
// and reflects the domain model used across the internal/attack and cmd/server
// packages.
package models

import (
	"sync"
	"time"
)

// ============================================================
// Enumerations (type aliases with constants)
// ============================================================

// SessionStatus represents the lifecycle state of a TestSession.
type SessionStatus string

const (
	StatusIdle      SessionStatus = "idle"
	StatusRunning   SessionStatus = "running"
	StatusCompleted SessionStatus = "completed"
	StatusStopped   SessionStatus = "stopped"
	StatusError     SessionStatus = "error"
)

// ErrorType classifies the cause of a failed HTTP request.
type ErrorType string

const (
	ErrorTypeTimeout           ErrorType = "timeout"
	ErrorTypeConnectionRefused ErrorType = "connection_refused"
	ErrorTypeDNS               ErrorType = "dns"
	ErrorTypeTLS               ErrorType = "tls"
	ErrorTypeOther             ErrorType = "other"
)

// ============================================================
// Package: attack — core domain types
// ============================================================

// AttackConfig holds all parameters required to run a load test.
type AttackConfig struct {
	URL         string
	Method      string
	Body        string
	Headers     []string
	Output      string
	Duration    time.Duration
	Timeout     time.Duration
	Workers     uint
	Connections uint
	RPS         int
}

// RequestResult is the outcome of a single HTTP request executed by a worker.
type RequestResult struct {
	StatusCode    int
	Latency       time.Duration
	Success       bool
	ErrorType     ErrorType
	BytesSent     uint64
	BytesReceived uint64
}

// LatencyPoint is a timestamped latency sample stored in the circular history
// buffer for real-time chart rendering.
type LatencyPoint struct {
	Timestamp time.Time
	Latency   time.Duration
}

// GlobalMetrics aggregates all measurements produced during a load test.
// All exported fields are safe to read after the test completes; during the
// test they must be accessed while holding the embedded Mutex.
type GlobalMetrics struct {
	TotalRequests uint64
	Successes     uint64
	Failures      uint64
	TotalLatency  time.Duration
	MinLatency    time.Duration
	MaxLatency    time.Duration
	BytesSent     uint64
	BytesReceived uint64

	ErrorTypes  map[ErrorType]uint64
	StatusCodes map[int]uint64

	// LatencyHistory is a circular buffer of sampled latency points used for
	// real-time chart data (max 1 000 entries, sampled every ~10 requests).
	LatencyHistory    []LatencyPoint
	latencyHistoryIdx int

	// Latencies is a circular buffer of raw latency samples used for
	// percentile calculation (max 100 000 entries).
	Latencies  []time.Duration
	latencyIdx int

	TargetRPS time.Duration
	StartTime time.Time
	Duration  time.Duration

	sync.Mutex
}

// NewGlobalMetrics allocates and initialises a GlobalMetrics with zero values.
func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		ErrorTypes:     make(map[ErrorType]uint64),
		StatusCodes:    make(map[int]uint64),
		LatencyHistory: make([]LatencyPoint, 0, 1_000),
		Latencies:      make([]time.Duration, 0, 100_000),
	}
}

// SetTestConfig stores the test configuration metadata needed for later report
// generation.
func (m *GlobalMetrics) SetTestConfig(targetRPS int, duration time.Duration, startTime time.Time) {
	m.Lock()
	defer m.Unlock()
	m.StartTime = startTime
	m.Duration = duration
}

// AddLatency records a single request latency sample.
func (m *GlobalMetrics) AddLatency(latency time.Duration) {
	m.Lock()
	defer m.Unlock()

	m.TotalLatency += latency
	if m.MinLatency == 0 || latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}

	// Circular buffer for percentile calculation (100 k entries).
	const maxLatencies = 100_000
	if len(m.Latencies) < maxLatencies {
		m.Latencies = append(m.Latencies, latency)
	} else {
		m.Latencies[m.latencyIdx%maxLatencies] = latency
		m.latencyIdx++
	}

	// Sparse sample for history chart (every ~10 requests).
	if m.TotalRequests%10 == 0 {
		const maxHistory = 1_000
		point := LatencyPoint{Timestamp: time.Now(), Latency: latency}
		if len(m.LatencyHistory) < maxHistory {
			m.LatencyHistory = append(m.LatencyHistory, point)
		} else {
			m.LatencyHistory[m.latencyHistoryIdx%maxHistory] = point
			m.latencyHistoryIdx++
		}
	}
}

// AddError increments the counter for the given error category.
func (m *GlobalMetrics) AddError(errType ErrorType) {
	m.Lock()
	defer m.Unlock()
	m.ErrorTypes[errType]++
}

// AddBytes accumulates network I/O byte counts.
func (m *GlobalMetrics) AddBytes(sent, received uint64) {
	m.Lock()
	defer m.Unlock()
	m.BytesSent += sent
	m.BytesReceived += received
}

// ============================================================
// Report types  (internal/attack/report.go)
// ============================================================

// AttackConfigJSON is the JSON-serialisable representation of AttackConfig.
type AttackConfigJSON struct {
	URL         string
	Method      string
	Body        string
	Headers     []string
	Duration    string
	Timeout     string
	Workers     uint
	Connections uint
	RPS         int
}

// Summary contains the high-level aggregated metrics for a finished test.
type Summary struct {
	TotalRequests      uint64
	SuccessfulRequests uint64
	FailedRequests     uint64
	TotalElapsedTime   string
	AverageLatency     string
	MinLatency         string
	MaxLatency         string
	ThroughputRPS      float64
	TargetRPS          int
	BytesSent          uint64
	BytesReceived      uint64
}

// LatencyPercentiles contains the P50/P90/P95/P99 latency values as formatted
// strings (e.g. "12.3ms").
type LatencyPercentiles struct {
	P50 string
	P90 string
	P95 string
	P99 string
}

// ErrorBreakdown maps each error category to its request count.
type ErrorBreakdown struct {
	Timeout           uint64
	ConnectionRefused uint64
	DNS               uint64
	TLS               uint64
	Other             uint64
}

// MetricsReport is the complete JSON report written to disk after a CLI test.
type MetricsReport struct {
	Config             AttackConfigJSON
	Summary            Summary
	LatencyPercentiles LatencyPercentiles
	StatusCodes        map[int]uint64
	ErrorBreakdown     ErrorBreakdown
	Timestamp          string
}

// ============================================================
// Package: server — HTTP/WebSocket server types
// ============================================================

// TestSession represents a single load-test run managed by the server.
type TestSession struct {
	ID          string
	Status      SessionStatus
	Protocol    string // "http" | "grpc"
	StartTime   time.Time
	EndTime     time.Time
	ElapsedTime time.Duration
	Error       error

	config  *AttackConfig
	metrics *GlobalMetrics
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// Start transitions the session from Idle to Running.
func (s *TestSession) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusRunning
	s.StartTime = time.Now()
}

// Stop signals the attack runner to stop and transitions status to Stopped.
func (s *TestSession) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Status == StatusRunning {
		close(s.stopCh)
		s.Status = StatusStopped
		s.EndTime = time.Now()
		s.ElapsedTime = s.EndTime.Sub(s.StartTime)
	}
}

// Complete stores the final metrics and marks the session as Completed.
func (s *TestSession) Complete(metrics *GlobalMetrics, elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = metrics
	s.ElapsedTime = elapsed
	s.EndTime = time.Now()
	s.Status = StatusCompleted
}

// SetError stores the error and transitions the session to the Error state.
func (s *TestSession) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
	s.Status = StatusError
}

// SetMetrics stores the metrics reference (used for live updates).
func (s *TestSession) SetMetrics(metrics *GlobalMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = metrics
}

// GetStatus returns the current session status.
func (s *TestSession) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// GetMetrics returns the current metrics snapshot (may be nil).
func (s *TestSession) GetMetrics() *GlobalMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

// IsRunning reports whether the session is actively executing.
func (s *TestSession) IsRunning() bool {
	return s.GetStatus() == StatusRunning
}

// SessionManager holds the single active TestSession.
type SessionManager struct {
	current *TestSession
	mu      sync.RWMutex
}

// NewSessionManager constructs an empty SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

// GetCurrent returns the current session (may be nil).
func (sm *SessionManager) GetCurrent() *TestSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// CreateSession stops any running session, then creates and stores a new HTTP
// session for the given config.
func (sm *SessionManager) CreateSession(id string, cfg *AttackConfig) *TestSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.current != nil && sm.current.IsRunning() {
		sm.current.Stop()
	}
	s := &TestSession{
		ID:       id,
		Status:   StatusIdle,
		Protocol: "http",
		config:   cfg,
		stopCh:   make(chan struct{}),
	}
	sm.current = s
	return s
}

// ============================================================
// WebSocket types (cmd/server/websocket.go)
// ============================================================

// WSMessage is the envelope for all messages sent over the WebSocket channel.
type WSMessage struct {
	Type      string      // "metrics" | "completed" | "stopped" | "error"
	Timestamp string      // RFC3339
	Data      any // MetricsPayload, or error string
}

// ============================================================
// DTO types for REST API
// ============================================================

// StartRequest is the body of POST /api/attack/start.
type StartRequest struct {
	Protocol     string
	Rate         string
	Duration     string
	Timeout      string
	Workers      uint
	Connections  uint
	URL          string
	Method       string
	Body         string
	Headers      []string
	GRPCTarget   string
	GRPCService  string
	GRPCMethod   string
	GRPCData     string
	GRPCProto    string
	GRPCMetadata []string
	Insecure     bool
	CACert       string
}

// StartResponse is returned by POST /api/attack/start on success.
type StartResponse struct {
	ID        string
	Status    string
	StartedAt string
}

// StopResponse is returned by POST /api/attack/stop on success.
type StopResponse struct {
	ID        string
	Status    string
	StoppedAt string
}

// LatencyPayload contains formatted latency percentile strings for API responses.
type LatencyPayload struct {
	P50 string
	P90 string
	P95 string
	P99 string
}

// ErrorBreakdownPayload contains error-category counts for API responses.
type ErrorBreakdownPayload struct {
	Timeout           uint64
	ConnectionRefused uint64
	DNS               uint64
	TLS               uint64
	Other             uint64
}

// LatencyHistoryPoint is a chart-ready data point used in MetricsPayload.
type LatencyHistoryPoint struct {
	Time    float64 // seconds since test start
	Latency float64 // latency in milliseconds
}

// MetricsPayload is the live metrics snapshot broadcast via WebSocket and
// returned by GET /api/attack/status.
type MetricsPayload struct {
	TotalRequests      uint64
	Successes          uint64
	Failures           uint64
	CurrentRPS         float64
	TargetRPS          int
	AverageLatency     string
	MinLatency         string
	MaxLatency         string
	LatencyPercentiles *LatencyPayload
	StatusCodes        map[int]uint64
	ErrorBreakdown     *ErrorBreakdownPayload
	BytesSent          uint64
	BytesReceived      uint64
	ElapsedTime        string
	Duration           string
	Progress           float64 // 0–100; -1 for infinite test
	LatencyHistory     []LatencyHistoryPoint
}

// StatusResponse is the body of GET /api/attack/status.
type StatusResponse struct {
	ID          string
	Status      string
	StartedAt   string
	ElapsedTime string
	Metrics     *MetricsPayload
}

// ConfigPayload contains test configuration included in the final results.
type ConfigPayload struct {
	URL         string
	Method      string
	Body        string
	Headers     []string
	Rate        string
	Duration    string
	Workers     uint
	Connections uint
}

// SummaryPayload is the high-level summary section of ResultsResponse.
type SummaryPayload struct {
	TotalRequests      uint64
	SuccessfulRequests uint64
	FailedRequests     uint64
	TotalElapsedTime   string
	AverageLatency     string
	MinLatency         string
	MaxLatency         string
	ThroughputRPS      float64
	TargetRPS          int
	BytesSent          uint64
	BytesReceived      uint64
}

// ResultsResponse is the body of GET /api/attack/results.
type ResultsResponse struct {
	ID                 string
	Status             string
	Config             *ConfigPayload
	Summary            *SummaryPayload
	LatencyPercentiles *LatencyPayload
	StatusCodes        map[int]uint64
	ErrorBreakdown     *ErrorBreakdownPayload
	Timestamp          string
}

// ErrorResponse is used for all non-2xx API responses.
type ErrorResponse struct {
	Error   string
	Message string
}

// ============================================================
// Interfaces
// ============================================================

// Runner is implemented by the attack package's RunAttack functions.
type Runner interface {
	RunAttack(cfg *AttackConfig, stopCh chan struct{}) (*GlobalMetrics, time.Duration, error)
	RunAttackWithMetrics(cfg *AttackConfig, stopCh chan struct{}, m *GlobalMetrics) (*GlobalMetrics, time.Duration, error)
}

// MetricsWriter is the write side of GlobalMetrics, used by worker goroutines.
type MetricsWriter interface {
	AddLatency(latency time.Duration)
	AddError(errType ErrorType)
	AddBytes(sent, received uint64)
}

// SessionStore abstracts the session persistence layer for the HTTP handlers.
type SessionStore interface {
	GetCurrent() *TestSession
	CreateSession(id string, cfg *AttackConfig) *TestSession
}
