/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/ratelimit"
)

// RunAttack executes the load test and returns metrics.
// stopCh: channel to signal test stop (close to stop)
// Returns: metrics, elapsed time, error
func RunAttack(cfg *AttackConfig, stopCh chan struct{}) (*GlobalMetrics, time.Duration, error) {
	return RunAttackWithMetrics(cfg, stopCh, nil)
}

// RunAttackWithMetrics executes the load test with an optional pre-created metrics object.
// If metrics is nil, a new one is created.
// This allows callers to monitor metrics in real-time during the attack.
func RunAttackWithMetrics(cfg *AttackConfig, stopCh chan struct{}, metrics *GlobalMetrics) (*GlobalMetrics, time.Duration, error) {
	client := createHTTPClient(cfg)
	if metrics == nil {
		metrics = NewGlobalMetrics()
	}

	startTime := time.Now()

	// Set test configuration for progress tracking
	metrics.SetTestConfig(cfg.RPS, cfg.Duration, startTime)

	jobsBufferSize := max(cfg.Workers*2, 100)
	jobs := make(chan struct{}, jobsBufferSize)
	results := make(chan RequestResult, cfg.Workers*2)
	var wg sync.WaitGroup

	for i := uint(1); i <= cfg.Workers; i++ {
		wg.Add(1)
		go worker(client, cfg, jobs, results, &wg)
	}

	rateLimiter := ratelimit.New(cfg.RPS)

	var stopOnce sync.Once
	stopFunc := func() {
		stopOnce.Do(func() {
			close(stopCh)
		})
	}

	var attackTimer *time.Timer
	if cfg.Duration > 0 {
		attackTimer = time.NewTimer(cfg.Duration)
		go func() {
			select {
			case <-attackTimer.C:
				stopFunc()
			case <-stopCh:
				attackTimer.Stop()
			}
		}()
	}

	go func() {
		defer close(jobs)

		for {
			select {
			case <-stopCh:
				return
			default:
				rateLimiter.Take()

				select {
				case <-stopCh:
					return
				case jobs <- struct{}{}:
				}
			}
		}
	}()

	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		defer collectorWG.Done()
		for res := range results {
			metrics.Lock()

			metrics.TotalRequests++
			metrics.TotalLatency += res.Latency
			if res.Success {
				metrics.Successes++
			} else {
				metrics.Failures++
				// Track error type
				if res.ErrorType != "" {
					metrics.AddError(res.ErrorType)
				}
			}
			metrics.AddLatency(res.Latency)
			metrics.StatusCodes[res.StatusCode]++

			// Track bytes
			metrics.AddBytes(res.BytesSent, res.BytesReceived)

			metrics.Unlock()
		}
	}()

	<-stopCh

	wg.Wait()
	close(results)
	collectorWG.Wait()

	elapsedTime := time.Since(startTime)
	if cfg.Duration > 0 && elapsedTime > cfg.Duration {
		elapsedTime = cfg.Duration
	}

	return metrics, elapsedTime, nil
}

func worker(
	client *http.Client,
	cfg *AttackConfig,
	jobs <-chan struct{},
	results chan<- RequestResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for range jobs {
		start := time.Now()
		result := RequestResult{Success: false}

		var bodyReader io.Reader
		var bodySize uint64
		if cfg.Body != "" {
			bodyReader = bytes.NewBufferString(cfg.Body)
			bodySize = uint64(len(cfg.Body))
		}

		req, err := http.NewRequest(cfg.Method, cfg.URL, bodyReader)
		if err != nil {
			result.ErrorType = ErrorTypeOther
			results <- result
			continue
		}

		for _, h := range cfg.Headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key != "" {
					req.Header.Set(key, value)
				}
			} else {
				fmt.Printf("Warning: Skipping invalid header format: %s\n", h)
			}
		}

		if (cfg.Method == "POST" || cfg.Method == "PUT") && cfg.Body != "" {
			if req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}
		}

		result.BytesSent = bodySize
		resp, err := client.Do(req)

		if err != nil {
			result.Latency = time.Since(start)
			result.StatusCode = 0
			result.ErrorType = classifyError(err)
			results <- result
			continue
		}

		// Read and discard response body to get size; latency includes full transfer
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		result.Latency = time.Since(start)
		result.BytesReceived = uint64(len(bodyBytes))
		result.StatusCode = resp.StatusCode
		if resp.StatusCode < 400 {
			result.Success = true
		}

		results <- result
	}
}

// classifyError determines the type of error that occurred
func classifyError(err error) ErrorType {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return ErrorTypeTimeout
	}

	// Check for connection refused
	if strings.Contains(errStr, "connection refused") {
		return ErrorTypeConnectionRefused
	}

	// Check for DNS errors
	if strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "lookup") ||
		strings.Contains(errStr, "dns") {
		return ErrorTypeDNS
	}

	// Check for TLS errors
	if _, ok := err.(*tls.CertificateVerificationError); ok {
		return ErrorTypeTLS
	}
	if strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "x509") {
		return ErrorTypeTLS
	}

	return ErrorTypeOther
}

func createHTTPClient(cfg *AttackConfig) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:      int(cfg.Connections),
		MaxConnsPerHost:   int(cfg.Connections),
		IdleConnTimeout:   90 * time.Second,
		DisableKeepAlives: false,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}
}
