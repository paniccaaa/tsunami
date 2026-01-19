/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
)

// StartRequest represents the request body for starting an attack
type StartRequest struct {
	URL         string   `json:"url"`
	Method      string   `json:"method"`
	Body        string   `json:"body"`
	Headers     []string `json:"headers"`
	Rate        string   `json:"rate"`
	Duration    string   `json:"duration"`
	Timeout     string   `json:"timeout"`
	Workers     uint     `json:"workers"`
	Connections uint     `json:"connections"`
}

// StartResponse represents the response for starting an attack
type StartResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

// StopResponse represents the response for stopping an attack
type StopResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StoppedAt string `json:"stopped_at"`
}

// StatusResponse represents the response for getting status
type StatusResponse struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"`
	StartedAt   string          `json:"started_at,omitempty"`
	ElapsedTime string          `json:"elapsed_time,omitempty"`
	Metrics     *MetricsPayload `json:"metrics,omitempty"`
}

// ErrorBreakdownPayload represents error type breakdown
type ErrorBreakdownPayload struct {
	Timeout          uint64 `json:"timeout"`
	ConnectionRefused uint64 `json:"connection_refused"`
	DNS              uint64 `json:"dns"`
	TLS              uint64 `json:"tls"`
	Other            uint64 `json:"other"`
}

// LatencyHistoryPoint represents a latency sample at a point in time
type LatencyHistoryPoint struct {
	Time    float64 `json:"time"`    // Seconds since test start
	Latency float64 `json:"latency"` // Latency in ms
}

// MetricsPayload represents real-time metrics
type MetricsPayload struct {
	TotalRequests      uint64          `json:"total_requests"`
	Successes          uint64          `json:"successes"`
	Failures           uint64          `json:"failures"`
	CurrentRPS         float64         `json:"current_rps"`
	TargetRPS          int             `json:"target_rps"`
	AverageLatency     string          `json:"average_latency"`
	MinLatency         string          `json:"min_latency"`
	MaxLatency         string          `json:"max_latency"`
	LatencyPercentiles *LatencyPayload `json:"latency_percentiles,omitempty"`
	StatusCodes        map[int]uint64  `json:"status_codes"`
	ErrorBreakdown     *ErrorBreakdownPayload `json:"error_breakdown,omitempty"`
	BytesSent          uint64          `json:"bytes_sent"`
	BytesReceived      uint64          `json:"bytes_received"`
	ElapsedTime        string          `json:"elapsed_time"`
	Duration           string          `json:"duration"`
	Progress           float64         `json:"progress"` // 0-100, -1 for infinite
	LatencyHistory     []LatencyHistoryPoint `json:"latency_history,omitempty"`
}

// LatencyPayload represents latency percentiles
type LatencyPayload struct {
	P50 string `json:"p50"`
	P90 string `json:"p90"`
	P95 string `json:"p95"`
	P99 string `json:"p99"`
}

// ResultsResponse represents the final results
type ResultsResponse struct {
	ID                 string                 `json:"id"`
	Status             string                 `json:"status"`
	Config             *ConfigPayload         `json:"config"`
	Summary            *SummaryPayload        `json:"summary"`
	LatencyPercentiles *LatencyPayload        `json:"latency_percentiles"`
	StatusCodes        map[int]uint64         `json:"status_codes"`
	ErrorBreakdown     *ErrorBreakdownPayload `json:"error_breakdown,omitempty"`
	Timestamp          string                 `json:"timestamp"`
}

// ConfigPayload represents the test configuration
type ConfigPayload struct {
	URL         string   `json:"url"`
	Method      string   `json:"method"`
	Body        string   `json:"body,omitempty"`
	Headers     []string `json:"headers,omitempty"`
	Rate        string   `json:"rate"`
	Duration    string   `json:"duration"`
	Workers     uint     `json:"workers"`
	Connections uint     `json:"connections"`
}

// SummaryPayload represents the test summary
type SummaryPayload struct {
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

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Handlers holds the HTTP handlers and dependencies
type Handlers struct {
	sessionManager *SessionManager
	wsHub          *Hub
}

// NewHandlers creates a new Handlers instance
func NewHandlers(sm *SessionManager, hub *Hub) *Handlers {
	return &Handlers{
		sessionManager: sm,
		wsHub:          hub,
	}
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, err, message string) {
	writeJSON(w, status, ErrorResponse{Error: err, Message: message})
}

// HandleStartAttack handles POST /api/attack/start
func (h *Handlers) HandleStartAttack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse request body")
		return
	}

	// Validate request
	if err := validateStartRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Parse duration
	var duration time.Duration
	if req.Duration != "" {
		var err error
		duration, err = time.ParseDuration(req.Duration)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_duration", "Invalid duration format")
			return
		}
	}

	// Parse timeout
	timeout := 10 * time.Second
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_timeout", "Invalid timeout format")
			return
		}
	}

	// Parse rate to RPS
	rps, err := parseRateToRPS(req.Rate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_rate", err.Error())
		return
	}

	// Set defaults
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Workers == 0 {
		req.Workers = 50
	}
	if req.Connections == 0 {
		req.Connections = 100
	}

	// Create attack config
	cfg := &attack.AttackConfig{
		URL:         req.URL,
		Method:      req.Method,
		Body:        req.Body,
		Headers:     req.Headers,
		Duration:    duration,
		Timeout:     timeout,
		Workers:     req.Workers,
		Connections: req.Connections,
		RPS:         rps,
	}

	// Generate session ID
	sessionID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	// Create session with pre-created metrics for real-time monitoring
	session := h.sessionManager.CreateSession(sessionID, cfg)
	session.SetMetrics(attack.NewGlobalMetrics())
	session.Start()

	// Start attack in goroutine
	go h.runAttack(session)

	// Start metrics streaming
	go h.streamMetrics(session)

	writeJSON(w, http.StatusOK, StartResponse{
		ID:        sessionID,
		Status:    string(StatusRunning),
		StartedAt: session.StartTime.Format(time.RFC3339),
	})
}

// runAttack runs the attack in a goroutine
func (h *Handlers) runAttack(session *TestSession) {
	metrics, elapsed, err := attack.RunAttackWithMetrics(session.Config, session.StopCh, session.GetMetrics())

	if err != nil {
		session.SetError(err)
		h.wsHub.Broadcast(WSMessage{
			Type:      "error",
			Timestamp: time.Now().Format(time.RFC3339),
			Data:      map[string]string{"error": err.Error()},
		})
		return
	}

	session.Complete(metrics, elapsed)
	h.wsHub.Broadcast(WSMessage{
		Type:      "completed",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      h.buildMetricsPayload(session),
	})
}

// streamMetrics streams metrics via WebSocket every 50ms for smooth UI updates
func (h *Handlers) streamMetrics(session *TestSession) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !session.IsRunning() {
				return
			}
			h.wsHub.Broadcast(WSMessage{
				Type:      "metrics",
				Timestamp: time.Now().Format(time.RFC3339),
				Data:      h.buildMetricsPayload(session),
			})
		case <-session.StopCh:
			return
		}
	}
}

// buildMetricsPayload creates a metrics payload from the session
func (h *Handlers) buildMetricsPayload(session *TestSession) *MetricsPayload {
	metrics := session.GetMetrics()
	if metrics == nil {
		return &MetricsPayload{
			StatusCodes: make(map[int]uint64),
			Progress:    -1,
		}
	}

	metrics.Lock()
	defer metrics.Unlock()

	elapsed := time.Since(session.StartTime)
	if session.ElapsedTime > 0 {
		elapsed = session.ElapsedTime
	}

	var currentRPS float64
	if elapsed.Seconds() > 0 {
		currentRPS = float64(metrics.TotalRequests) / elapsed.Seconds()
	}

	var avgLatency string
	if metrics.TotalRequests > 0 {
		avg := metrics.TotalLatency / time.Duration(metrics.TotalRequests)
		avgLatency = avg.Round(time.Millisecond).String()
	}

	// Min/Max latency
	var minLatency, maxLatency string
	if metrics.MinLatency < time.Duration(1<<63-1) {
		minLatency = metrics.MinLatency.Round(time.Millisecond).String()
	} else {
		minLatency = "0ms"
	}
	maxLatency = metrics.MaxLatency.Round(time.Millisecond).String()

	// Progress calculation
	var progress float64 = -1 // -1 means infinite
	duration := session.Config.Duration
	if duration > 0 {
		progress = (elapsed.Seconds() / duration.Seconds()) * 100
		if progress > 100 {
			progress = 100
		}
	}

	payload := &MetricsPayload{
		TotalRequests:  metrics.TotalRequests,
		Successes:      metrics.Successes,
		Failures:       metrics.Failures,
		CurrentRPS:     currentRPS,
		TargetRPS:      session.Config.RPS,
		AverageLatency: avgLatency,
		MinLatency:     minLatency,
		MaxLatency:     maxLatency,
		StatusCodes:    make(map[int]uint64),
		BytesSent:      metrics.BytesSent,
		BytesReceived:  metrics.BytesReceived,
		ElapsedTime:    elapsed.Round(time.Second).String(),
		Duration:       duration.String(),
		Progress:       progress,
	}

	// Copy status codes
	for code, count := range metrics.StatusCodes {
		payload.StatusCodes[code] = count
	}

	// Error breakdown
	if metrics.Failures > 0 {
		payload.ErrorBreakdown = &ErrorBreakdownPayload{
			Timeout:          metrics.ErrorTypes[attack.ErrorTypeTimeout],
			ConnectionRefused: metrics.ErrorTypes[attack.ErrorTypeConnectionRefused],
			DNS:              metrics.ErrorTypes[attack.ErrorTypeDNS],
			TLS:              metrics.ErrorTypes[attack.ErrorTypeTLS],
			Other:            metrics.ErrorTypes[attack.ErrorTypeOther],
		}
	}

	// Calculate percentiles if we have data
	if len(metrics.Latencies) > 0 {
		payload.LatencyPercentiles = &LatencyPayload{
			P50: attack.CalculatePercentile(metrics.Latencies, 50).Round(time.Millisecond).String(),
			P90: attack.CalculatePercentile(metrics.Latencies, 90).Round(time.Millisecond).String(),
			P95: attack.CalculatePercentile(metrics.Latencies, 95).Round(time.Millisecond).String(),
			P99: attack.CalculatePercentile(metrics.Latencies, 99).Round(time.Millisecond).String(),
		}
	}

	// Build latency history for chart
	if len(metrics.LatencyHistory) > 0 {
		payload.LatencyHistory = make([]LatencyHistoryPoint, 0, len(metrics.LatencyHistory))
		for _, point := range metrics.LatencyHistory {
			payload.LatencyHistory = append(payload.LatencyHistory, LatencyHistoryPoint{
				Time:    point.Timestamp.Sub(session.StartTime).Seconds(),
				Latency: float64(point.Latency.Milliseconds()),
			})
		}
	}

	return payload
}

// HandleStopAttack handles POST /api/attack/stop
func (h *Handlers) HandleStopAttack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	session := h.sessionManager.GetCurrent()
	if session == nil {
		writeError(w, http.StatusNotFound, "no_session", "No test session found")
		return
	}

	if session.GetStatus() != StatusRunning {
		writeError(w, http.StatusConflict, "not_running", "No test is currently running")
		return
	}

	session.Stop()

	h.wsHub.Broadcast(WSMessage{
		Type:      "stopped",
		Timestamp: time.Now().Format(time.RFC3339),
	})

	writeJSON(w, http.StatusOK, StopResponse{
		ID:        session.ID,
		Status:    string(StatusStopped),
		StoppedAt: time.Now().Format(time.RFC3339),
	})
}

// HandleStatus handles GET /api/attack/status
func (h *Handlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	session := h.sessionManager.GetCurrent()
	if session == nil {
		writeJSON(w, http.StatusOK, StatusResponse{
			Status: string(StatusIdle),
		})
		return
	}

	response := StatusResponse{
		ID:        session.ID,
		Status:    string(session.GetStatus()),
		StartedAt: session.StartTime.Format(time.RFC3339),
	}

	if session.GetStatus() == StatusRunning {
		response.ElapsedTime = time.Since(session.StartTime).Round(time.Second).String()
		response.Metrics = h.buildMetricsPayload(session)
	} else if session.GetStatus() == StatusCompleted || session.GetStatus() == StatusStopped {
		response.ElapsedTime = session.ElapsedTime.Round(time.Second).String()
		response.Metrics = h.buildMetricsPayload(session)
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleResults handles GET /api/attack/results
func (h *Handlers) HandleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	session := h.sessionManager.GetCurrent()
	if session == nil {
		writeError(w, http.StatusNotFound, "no_session", "No test session found")
		return
	}

	status := session.GetStatus()
	if status != StatusCompleted && status != StatusStopped {
		writeError(w, http.StatusConflict, "test_not_finished", "Test is still running or not started")
		return
	}

	metrics := session.GetMetrics()
	if metrics == nil {
		writeError(w, http.StatusNotFound, "no_results", "No results available")
		return
	}

	response := h.buildResultsResponse(session)
	writeJSON(w, http.StatusOK, response)
}

// HandleDownload handles GET /api/attack/results/download
func (h *Handlers) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	session := h.sessionManager.GetCurrent()
	if session == nil {
		writeError(w, http.StatusNotFound, "no_session", "No test session found")
		return
	}

	status := session.GetStatus()
	if status != StatusCompleted && status != StatusStopped {
		writeError(w, http.StatusConflict, "test_not_finished", "Test is still running or not started")
		return
	}

	response := h.buildResultsResponse(session)

	filename := fmt.Sprintf("tsunami-results-%s.json", time.Now().Format("2006-01-02T15-04-05Z"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	json.NewEncoder(w).Encode(response)
}

// buildResultsResponse builds the results response
func (h *Handlers) buildResultsResponse(session *TestSession) *ResultsResponse {
	metrics := session.GetMetrics()
	if metrics == nil {
		return nil
	}

	metrics.Lock()
	defer metrics.Unlock()

	elapsed := session.ElapsedTime
	var throughput float64
	if elapsed.Seconds() > 0 {
		throughput = float64(metrics.TotalRequests) / elapsed.Seconds()
	}

	var avgLatency string
	if metrics.TotalRequests > 0 {
		avg := metrics.TotalLatency / time.Duration(metrics.TotalRequests)
		avgLatency = avg.Round(time.Millisecond).String()
	}

	// Min/Max latency
	var minLatency, maxLatency string
	if metrics.MinLatency < time.Duration(1<<63-1) {
		minLatency = metrics.MinLatency.Round(time.Millisecond).String()
	} else {
		minLatency = "0ms"
	}
	maxLatency = metrics.MaxLatency.Round(time.Millisecond).String()

	cfg := session.Config
	response := &ResultsResponse{
		ID:     session.ID,
		Status: string(session.GetStatus()),
		Config: &ConfigPayload{
			URL:         cfg.URL,
			Method:      cfg.Method,
			Body:        cfg.Body,
			Headers:     cfg.Headers,
			Rate:        fmt.Sprintf("%d/1s", cfg.RPS),
			Duration:    cfg.Duration.String(),
			Workers:     cfg.Workers,
			Connections: cfg.Connections,
		},
		Summary: &SummaryPayload{
			TotalRequests:      metrics.TotalRequests,
			SuccessfulRequests: metrics.Successes,
			FailedRequests:     metrics.Failures,
			TotalElapsedTime:   elapsed.Round(time.Second).String(),
			AverageLatency:     avgLatency,
			MinLatency:         minLatency,
			MaxLatency:         maxLatency,
			ThroughputRPS:      throughput,
			TargetRPS:          cfg.RPS,
			BytesSent:          metrics.BytesSent,
			BytesReceived:      metrics.BytesReceived,
		},
		StatusCodes: make(map[int]uint64),
		Timestamp:   session.EndTime.Format(time.RFC3339),
	}

	// Copy status codes
	for code, count := range metrics.StatusCodes {
		response.StatusCodes[code] = count
	}

	// Error breakdown
	if metrics.Failures > 0 {
		response.ErrorBreakdown = &ErrorBreakdownPayload{
			Timeout:          metrics.ErrorTypes[attack.ErrorTypeTimeout],
			ConnectionRefused: metrics.ErrorTypes[attack.ErrorTypeConnectionRefused],
			DNS:              metrics.ErrorTypes[attack.ErrorTypeDNS],
			TLS:              metrics.ErrorTypes[attack.ErrorTypeTLS],
			Other:            metrics.ErrorTypes[attack.ErrorTypeOther],
		}
	}

	// Calculate percentiles
	if len(metrics.Latencies) > 0 {
		response.LatencyPercentiles = &LatencyPayload{
			P50: attack.CalculatePercentile(metrics.Latencies, 50).Round(time.Millisecond).String(),
			P90: attack.CalculatePercentile(metrics.Latencies, 90).Round(time.Millisecond).String(),
			P95: attack.CalculatePercentile(metrics.Latencies, 95).Round(time.Millisecond).String(),
			P99: attack.CalculatePercentile(metrics.Latencies, 99).Round(time.Millisecond).String(),
		}
	}

	return response
}

// Validation helpers

var rateRegex = regexp.MustCompile(`^(\d+)/(\d+)(ms|s|m|h)$`)

func validateStartRequest(req *StartRequest) error {
	if req.URL == "" {
		return fmt.Errorf("url is required")
	}

	u, err := url.ParseRequestURI(req.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %s", err.Error())
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must have http or https scheme")
	}

	if req.Rate == "" {
		req.Rate = "100/1s"
	}

	if !rateRegex.MatchString(req.Rate) {
		return fmt.Errorf("invalid rate format, expected: NUMBER/TIME (e.g., 100/1s)")
	}

	return nil
}

func parseRateToRPS(rate string) (int, error) {
	matches := rateRegex.FindStringSubmatch(rate)
	if len(matches) != 4 {
		return 0, fmt.Errorf("invalid rate format: %s", rate)
	}

	var requests int
	fmt.Sscanf(matches[1], "%d", &requests)

	var value int
	fmt.Sscanf(matches[2], "%d", &value)

	unit := matches[3]

	var interval time.Duration
	switch unit {
	case "ms":
		interval = time.Millisecond * time.Duration(value)
	case "s":
		interval = time.Second * time.Duration(value)
	case "m":
		interval = time.Minute * time.Duration(value)
	case "h":
		interval = time.Hour * time.Duration(value)
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}

	rps := float64(requests) / interval.Seconds()
	if rps < 1 {
		return 1, nil
	}

	return int(rps), nil
}
