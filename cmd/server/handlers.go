package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
	"github.com/paniccaaa/tsunami/internal/grpcattack"
)

type StartRequest struct {
	Protocol    string        `json:"protocol"`
	Rate        string        `json:"rate"`
	Duration    string        `json:"duration"`
	Timeout     string        `json:"timeout"`
	Workers     uint          `json:"workers"`
	Connections uint          `json:"connections"`

	URL         string        `json:"url"`
	Method      string        `json:"method"`
	Body        string        `json:"body"`
	Headers     []string      `json:"headers"`

	GRPCTarget   string   `json:"grpc_target"`
	GRPCService  string   `json:"grpc_service"`
	GRPCMethod   string   `json:"grpc_method"`
	GRPCData     string   `json:"grpc_data"`
	GRPCProto    string   `json:"grpc_proto"`
	GRPCMetadata []string `json:"grpc_metadata"`
	Insecure     bool     `json:"insecure"`
	CACert       string   `json:"ca_cert"`
}

type StartResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

type StopResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StoppedAt string `json:"stopped_at"`
}

type StatusResponse struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"`
	StartedAt   string          `json:"started_at,omitempty"`
	ElapsedTime string          `json:"elapsed_time,omitempty"`
	Metrics     *MetricsPayload `json:"metrics,omitempty"`
}

type ErrorBreakdownPayload struct {
	Timeout          uint64 `json:"timeout"`
	ConnectionRefused uint64 `json:"connection_refused"`
	DNS              uint64 `json:"dns"`
	TLS              uint64 `json:"tls"`
	Other            uint64 `json:"other"`
}

type LatencyHistoryPoint struct {
	Time    float64 `json:"time"`
	Latency float64 `json:"latency"`
}

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
	Progress           float64         `json:"progress"`
	LatencyHistory     []LatencyHistoryPoint `json:"latency_history,omitempty"`
}

type LatencyPayload struct {
	P50 string `json:"p50"`
	P90 string `json:"p90"`
	P95 string `json:"p95"`
	P99 string `json:"p99"`
}

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

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type UploadProtoResponse struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type Handlers struct {
	sessionManager *SessionManager
	wsHub          *Hub
}

func NewHandlers(sm *SessionManager, hub *Hub) *Handlers {
	return &Handlers{
		sessionManager: sm,
		wsHub:          hub,
	}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:3000" || origin == "http://localhost:8080" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, err, message string) {
	writeJSON(w, status, ErrorResponse{Error: err, Message: message})
}

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

	if err := validateStartRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	var duration time.Duration
	if req.Duration != "" {
		var err error
		duration, err = time.ParseDuration(req.Duration)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_duration", "Invalid duration format")
			return
		}
	}

	timeout := 10 * time.Second
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_timeout", "Invalid timeout format")
			return
		}
	}

	rps, err := attack.ParseRateToRPS(req.Rate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_rate", err.Error())
		return
	}

	if req.Workers == 0 {
		req.Workers = 50
	}

	sessionID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	if req.Protocol == "grpc" {
		if req.GRPCTarget == "" || req.GRPCService == "" || req.GRPCMethod == "" {
			writeError(w, http.StatusBadRequest, "validation_error",
				"grpc_target, grpc_service and grpc_method are required for gRPC tests")
			return
		}
		if req.Connections == 0 {
			req.Connections = 4
		}
		grpcCfg := &grpcattack.Config{
			Target:      req.GRPCTarget,
			Service:     req.GRPCService,
			Method:      req.GRPCMethod,
			Data:        req.GRPCData,
			ProtoFile:   req.GRPCProto,
			Metadata:    req.GRPCMetadata,
			Insecure:    req.Insecure,
			CACert:      req.CACert,
			Duration:    duration,
			Timeout:     timeout,
			Workers:     req.Workers,
			Connections: req.Connections,
			RPS:         rps,
		}

		session := h.sessionManager.CreateGRPCSession(sessionID, grpcCfg)
		session.SetMetrics(attack.NewGlobalMetrics())
		session.Start()

		go h.runGRPCAttack(session)
		go h.streamMetrics(session)

		writeJSON(w, http.StatusOK, StartResponse{
			ID:        sessionID,
			Status:    string(StatusRunning),
			StartedAt: session.StartTime.Format(time.RFC3339),
		})
		return
	}

	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Connections == 0 {
		req.Connections = 100
	}

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

	session := h.sessionManager.CreateSession(sessionID, cfg)
	session.SetMetrics(attack.NewGlobalMetrics())
	session.Start()

	go h.runAttack(session)
	go h.streamMetrics(session)

	writeJSON(w, http.StatusOK, StartResponse{
		ID:        sessionID,
		Status:    string(StatusRunning),
		StartedAt: session.StartTime.Format(time.RFC3339),
	})
}

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

func (h *Handlers) runGRPCAttack(session *TestSession) {
	metrics, elapsed, err := grpcattack.RunAttack(session.GRPCConfig, session.StopCh, session.GetMetrics())

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

	var minLatency, maxLatency string
	if metrics.MinLatency < time.Duration(1<<63-1) {
		minLatency = metrics.MinLatency.Round(time.Millisecond).String()
	} else {
		minLatency = "0ms"
	}
	maxLatency = metrics.MaxLatency.Round(time.Millisecond).String()

	var duration time.Duration
	var targetRPS int
	if session.Config != nil {
		duration = session.Config.Duration
		targetRPS = session.Config.RPS
	} else if session.GRPCConfig != nil {
		duration = session.GRPCConfig.Duration
		targetRPS = session.GRPCConfig.RPS
	}

	var progress float64 = -1
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
		TargetRPS:      targetRPS,
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

	for code, count := range metrics.StatusCodes {
		payload.StatusCodes[code] = count
	}

	if metrics.Failures > 0 {
		payload.ErrorBreakdown = &ErrorBreakdownPayload{
			Timeout:          metrics.ErrorTypes[attack.ErrorTypeTimeout],
			ConnectionRefused: metrics.ErrorTypes[attack.ErrorTypeConnectionRefused],
			DNS:              metrics.ErrorTypes[attack.ErrorTypeDNS],
			TLS:              metrics.ErrorTypes[attack.ErrorTypeTLS],
			Other:            metrics.ErrorTypes[attack.ErrorTypeOther],
		}
	}

	if len(metrics.Latencies) > 0 {
		p50, p90, p95, p99 := attack.CalculateAllPercentiles(metrics.Latencies)
		payload.LatencyPercentiles = &LatencyPayload{
			P50: p50.Round(time.Millisecond).String(),
			P90: p90.Round(time.Millisecond).String(),
			P95: p95.Round(time.Millisecond).String(),
			P99: p99.Round(time.Millisecond).String(),
		}
	}

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

	var minLatency, maxLatency string
	if metrics.MinLatency < time.Duration(1<<63-1) {
		minLatency = metrics.MinLatency.Round(time.Millisecond).String()
	} else {
		minLatency = "0ms"
	}
	maxLatency = metrics.MaxLatency.Round(time.Millisecond).String()

	var cfgPayload *ConfigPayload
	var summaryRPS int
	if session.Config != nil {
		cfg := session.Config
		summaryRPS = cfg.RPS
		cfgPayload = &ConfigPayload{
			URL:         cfg.URL,
			Method:      cfg.Method,
			Body:        cfg.Body,
			Headers:     cfg.Headers,
			Rate:        fmt.Sprintf("%d/1s", cfg.RPS),
			Duration:    cfg.Duration.String(),
			Workers:     cfg.Workers,
			Connections: cfg.Connections,
		}
	} else if session.GRPCConfig != nil {
		gcfg := session.GRPCConfig
		summaryRPS = gcfg.RPS
		cfgPayload = &ConfigPayload{
			URL:         gcfg.Target,
			Method:      gcfg.Service + "/" + gcfg.Method,
			Rate:        fmt.Sprintf("%d/1s", gcfg.RPS),
			Duration:    gcfg.Duration.String(),
			Workers:     gcfg.Workers,
			Connections: gcfg.Connections,
		}
	}

	response := &ResultsResponse{
		ID:     session.ID,
		Status: string(session.GetStatus()),
		Config: cfgPayload,
		Summary: &SummaryPayload{
			TotalRequests:      metrics.TotalRequests,
			SuccessfulRequests: metrics.Successes,
			FailedRequests:     metrics.Failures,
			TotalElapsedTime:   elapsed.Round(time.Second).String(),
			AverageLatency:     avgLatency,
			MinLatency:         minLatency,
			MaxLatency:         maxLatency,
			ThroughputRPS:      throughput,
			TargetRPS:          summaryRPS,
			BytesSent:          metrics.BytesSent,
			BytesReceived:      metrics.BytesReceived,
		},
		StatusCodes: make(map[int]uint64),
		Timestamp:   session.EndTime.Format(time.RFC3339),
	}

	for code, count := range metrics.StatusCodes {
		response.StatusCodes[code] = count
	}

	if metrics.Failures > 0 {
		response.ErrorBreakdown = &ErrorBreakdownPayload{
			Timeout:          metrics.ErrorTypes[attack.ErrorTypeTimeout],
			ConnectionRefused: metrics.ErrorTypes[attack.ErrorTypeConnectionRefused],
			DNS:              metrics.ErrorTypes[attack.ErrorTypeDNS],
			TLS:              metrics.ErrorTypes[attack.ErrorTypeTLS],
			Other:            metrics.ErrorTypes[attack.ErrorTypeOther],
		}
	}

	if len(metrics.Latencies) > 0 {
		p50, p90, p95, p99 := attack.CalculateAllPercentiles(metrics.Latencies)
		response.LatencyPercentiles = &LatencyPayload{
			P50: p50.Round(time.Millisecond).String(),
			P90: p90.Round(time.Millisecond).String(),
			P95: p95.Round(time.Millisecond).String(),
			P99: p99.Round(time.Millisecond).String(),
		}
	}

	return response
}

func (h *Handlers) HandleProtoUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_form", "Failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "No file provided in 'file' field")
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".proto") {
		writeError(w, http.StatusBadRequest, "invalid_file_type", "Only .proto files are accepted")
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_error", "Failed to read uploaded file")
		return
	}

	tmpFile, err := os.CreateTemp("", "tsunami-proto-*.proto")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "temp_file_error", "Failed to create temp file")
		return
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		os.Remove(tmpFile.Name())
		writeError(w, http.StatusInternalServerError, "write_error", "Failed to write temp file")
		return
	}

	writeJSON(w, http.StatusOK, UploadProtoResponse{
		Path: tmpFile.Name(),
		Name: header.Filename,
	})
}

func validateStartRequest(req *StartRequest) error {
	if req.Protocol != "grpc" {
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
	}

	if req.Rate == "" {
		req.Rate = "100/1s"
	}

	if !attack.RateRegex.MatchString(req.Rate) {
		return fmt.Errorf("invalid rate format, expected: NUMBER/TIME (e.g., 100/1s)")
	}

	return nil
}
