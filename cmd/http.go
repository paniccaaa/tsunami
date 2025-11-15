package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var defaultHeaders = []string{}

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

	RPS int
}

type RequestResult struct {
	StatusCode int
	Latency    time.Duration
	Success    bool
}

func GetAttackConfig(cmd *cobra.Command) (*AttackConfig, error) {
	url, _ := cmd.Flags().GetString("url")
	method, _ := cmd.Flags().GetString("method")
	body, _ := cmd.Flags().GetString("body")
	headers, _ := cmd.Flags().GetStringArray("headers")
	output, _ := cmd.Flags().GetString("output")

	duration, _ := cmd.Flags().GetDuration("duration")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	workers, _ := cmd.Flags().GetUint("workers")
	connections, _ := cmd.Flags().GetUint("connections")

	rateStr, _ := cmd.Flags().GetString("rate")

	rps, err := parseRateToRPS(rateStr)
	if err != nil {
		return nil, err
	}

	return &AttackConfig{
		URL:         url,
		Method:      method,
		Body:        body,
		Headers:     headers,
		Output:      output,
		Duration:    duration,
		Timeout:     timeout,
		Workers:     workers,
		Connections: connections,
		RPS:         rps,
	}, nil
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
