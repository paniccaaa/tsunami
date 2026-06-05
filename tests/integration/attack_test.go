//go:build integration

package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
)

// TestRunAttack_BasicSuccess checks that a normal attack completes without errors:
// all requests get 200, failures are zero, and latency is recorded.
func TestRunAttack_BasicSuccess(t *testing.T) {
	cfg := &attack.AttackConfig{
		URL:         testBaseURL + "/get",
		Method:      "GET",
		RPS:         10,
		Duration:    2 * time.Second,
		Workers:     5,
		Connections: 10,
		Timeout:     5 * time.Second,
	}

	stopCh := make(chan struct{})
	metrics, elapsed, err := attack.RunAttack(cfg, stopCh)
	if err != nil {
		t.Fatalf("RunAttack failed: %v", err)
	}

	if metrics.TotalRequests == 0 {
		t.Error("expected TotalRequests > 0")
	}
	if metrics.Failures > 0 {
		t.Errorf("expected 0 failures, got %d", metrics.Failures)
	}
	if metrics.StatusCodes[200] == 0 {
		t.Error("expected 200 status codes to be counted")
	}
	if metrics.TotalRequests != metrics.Successes {
		t.Errorf("all requests should be successful: total=%d successes=%d",
			metrics.TotalRequests, metrics.Successes)
	}
	if elapsed == 0 {
		t.Error("expected elapsed time > 0")
	}
	if metrics.MaxLatency == 0 {
		t.Error("expected MaxLatency to be recorded")
	}
	if metrics.MinLatency >= metrics.MaxLatency && metrics.TotalRequests > 1 {
		t.Errorf("expected MinLatency < MaxLatency, got min=%v max=%v",
			metrics.MinLatency, metrics.MaxLatency)
	}
}

// TestRunAttack_TimeoutClassification checks that a slow server produces
// timeout errors and nothing else.
// /delay/3 makes the server wait 3s; our timeout is 500ms so every request times out.
func TestRunAttack_TimeoutClassification(t *testing.T) {
	cfg := &attack.AttackConfig{
		URL:         testBaseURL + "/delay/3",
		Method:      "GET",
		RPS:         5,
		Duration:    2 * time.Second,
		Workers:     5,
		Connections: 10,
		Timeout:     500 * time.Millisecond, // shorter than the server delay
	}

	stopCh := make(chan struct{})
	metrics, _, err := attack.RunAttack(cfg, stopCh)
	if err != nil {
		t.Fatalf("RunAttack failed: %v", err)
	}

	if metrics.Failures == 0 {
		t.Error("expected failures due to timeout, got 0")
	}
	if metrics.Successes > 0 {
		t.Errorf("expected 0 successes for slow endpoint, got %d", metrics.Successes)
	}
	if metrics.ErrorTypes[attack.ErrorTypeTimeout] == 0 {
		t.Errorf("expected ErrorTypeTimeout, got: %v", metrics.ErrorTypes)
	}
	// Make sure no other error types sneak in
	for errType, count := range metrics.ErrorTypes {
		if errType != attack.ErrorTypeTimeout && count > 0 {
			t.Errorf("unexpected error type %q: %d", errType, count)
		}
	}
}

// TestRunAttack_MetricsAccuracy checks that Successes and Failures
// are counted correctly based on HTTP status code.
// Rule in runner.go: status < 400 → success, status >= 400 → failure.
func TestRunAttack_MetricsAccuracy(t *testing.T) {
	t.Run("2xx is a success", func(t *testing.T) {
		cfg := &attack.AttackConfig{
			URL:         testBaseURL + "/status/201",
			Method:      "GET",
			RPS:         10,
			Duration:    2 * time.Second,
			Workers:     5,
			Connections: 10,
			Timeout:     5 * time.Second,
		}

		stopCh := make(chan struct{})
		metrics, _, err := attack.RunAttack(cfg, stopCh)
		if err != nil {
			t.Fatalf("RunAttack failed: %v", err)
		}

		if metrics.Successes == 0 {
			t.Error("expected Successes > 0 for /status/201")
		}
		if metrics.Failures > 0 {
			t.Errorf("expected 0 Failures for /status/201, got %d", metrics.Failures)
		}
		if metrics.StatusCodes[201] == 0 {
			t.Error("expected 201 to be tracked in StatusCodes")
		}
	})

	t.Run("5xx is a failure", func(t *testing.T) {
		cfg := &attack.AttackConfig{
			URL:         testBaseURL + "/status/500",
			Method:      "GET",
			RPS:         10,
			Duration:    2 * time.Second,
			Workers:     5,
			Connections: 10,
			Timeout:     5 * time.Second,
		}

		stopCh := make(chan struct{})
		metrics, _, err := attack.RunAttack(cfg, stopCh)
		if err != nil {
			t.Fatalf("RunAttack failed: %v", err)
		}

		if metrics.Failures == 0 {
			t.Error("expected Failures > 0 for /status/500")
		}
		if metrics.Successes > 0 {
			t.Errorf("expected 0 Successes for /status/500, got %d", metrics.Successes)
		}
		if metrics.StatusCodes[500] == 0 {
			t.Error("expected 500 to be tracked in StatusCodes")
		}
		// HTTP 500 is a Failure but not a network error — ErrorTypes must stay empty
		if len(metrics.ErrorTypes) > 0 {
			t.Errorf("HTTP 500 should not produce ErrorType entries, got: %v", metrics.ErrorTypes)
		}
	})

	t.Run("4xx is a failure", func(t *testing.T) {
		cfg := &attack.AttackConfig{
			URL:         testBaseURL + "/status/404",
			Method:      "GET",
			RPS:         10,
			Duration:    2 * time.Second,
			Workers:     5,
			Connections: 10,
			Timeout:     5 * time.Second,
		}

		stopCh := make(chan struct{})
		metrics, _, err := attack.RunAttack(cfg, stopCh)
		if err != nil {
			t.Fatalf("RunAttack failed: %v", err)
		}

		if metrics.Failures == 0 {
			t.Error("expected Failures > 0 for /status/404")
		}
		if metrics.StatusCodes[404] == 0 {
			t.Error("expected 404 to be tracked in StatusCodes")
		}
	})
}

// TestHighThroughput verifies the engine sustains ≥ 1 000 RPS with P99 ≤ 50 ms.
// Uses a minimal in-process HTTP server to measure engine throughput without Docker overhead.
func TestHighThroughput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &attack.AttackConfig{
		URL:         srv.URL,
		Method:      "GET",
		RPS:         1000,
		Duration:    10 * time.Second,
		Workers:     50,
		Connections: 100,
		Timeout:     5 * time.Second,
	}

	stopCh := make(chan struct{})
	metrics, elapsed, err := attack.RunAttack(cfg, stopCh)
	if err != nil {
		t.Fatalf("RunAttack failed: %v", err)
	}

	actualRPS := float64(metrics.TotalRequests) / elapsed.Seconds()
	if actualRPS < 1000 {
		t.Errorf("throughput %.0f RPS is below target 1000 RPS", actualRPS)
	}
	if metrics.Failures > 0 {
		t.Errorf("expected 0 failures, got %d", metrics.Failures)
	}
	_, _, _, p99 := attack.CalculateAllPercentiles(metrics.Latencies)
	if p99 > 50*time.Millisecond {
		t.Errorf("P99 %v exceeds 50 ms threshold", p99)
	}
}
