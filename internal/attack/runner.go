/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"bytes"
	"fmt"
	"io"
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

	jobsBufferSize := cfg.Workers * 2
	if jobsBufferSize < 100 {
		jobsBufferSize = 100
	}
	jobs := make(chan struct{}, jobsBufferSize)
	results := make(chan RequestResult, cfg.Workers*2)
	var wg sync.WaitGroup

	for i := uint(1); i <= cfg.Workers; i++ {
		wg.Add(1)
		go worker(client, cfg, jobs, results, &wg)
	}

	rateLimiter := ratelimit.New(cfg.RPS)

	startTime := time.Now()

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
			}
			metrics.AddLatency(res.Latency)
			metrics.StatusCodes[res.StatusCode]++

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
		if cfg.Body != "" {
			bodyReader = bytes.NewBufferString(cfg.Body)
		}

		req, err := http.NewRequest(cfg.Method, cfg.URL, bodyReader)
		if err != nil {
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
		resp, err := client.Do(req)

		latency := time.Since(start)
		result.Latency = latency

		if err != nil {
			result.StatusCode = 0
			results <- result
			continue
		}

		result.StatusCode = resp.StatusCode
		if resp.StatusCode < 400 {
			result.Success = true
		}

		results <- result
		resp.Body.Close()
	}
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
