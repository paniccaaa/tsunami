/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package cmd

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
	"github.com/spf13/cobra"
)

const (
	defaultDuration        = 0
	defaultTimeoutDuration = time.Second * 10

	defaultRate = "100/1s"

	defaultWorkers    = 50
	defaultMaxWorkers = math.MaxUint

	defaultConnections    = 100
	defaultMaxConnections = math.MaxUint

	defaultMethod = "GET"
	defaultOutput = "stdout"
	defaultBody   = ""
)

var defaultHeaders = []string{}

// Spinner frames for live progress
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatDuration formats duration in a compact way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// liveProgress displays real-time metrics during the attack.
// It returns a channel that is closed when the goroutine exits.
func liveProgress(metrics *attack.GlobalMetrics, cfg *attack.AttackConfig, startTime time.Time, stopCh chan struct{}) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		frameIdx := 0
		lastRequests := uint64(0)
		lastTime := startTime

		for {
			select {
			case <-stopCh:
				// Clear the line and return
				fmt.Print("\r\033[K")
				return
			case <-ticker.C:
			metrics.Lock()

			elapsed := time.Since(startTime)

			// Calculate current RPS (based on recent requests)
			now := time.Now()
			timeDelta := now.Sub(lastTime).Seconds()
			requestDelta := metrics.TotalRequests - lastRequests
			var currentRPS float64
			if timeDelta > 0 {
				currentRPS = float64(requestDelta) / timeDelta
			}
			lastRequests = metrics.TotalRequests
			lastTime = now

			// Calculate average RPS
			var avgRPS float64
			if elapsed.Seconds() > 0 {
				avgRPS = float64(metrics.TotalRequests) / elapsed.Seconds()
			}

			// Calculate average latency
			var avgLatency time.Duration
			if metrics.TotalRequests > 0 {
				avgLatency = metrics.TotalLatency / time.Duration(metrics.TotalRequests)
			}

			// Progress bar for duration
			var progressBar string
			if cfg.Duration > 0 {
				progress := elapsed.Seconds() / cfg.Duration.Seconds()
				if progress > 1 {
					progress = 1
				}
				barWidth := 20
				filled := int(progress * float64(barWidth))
				progressBar = fmt.Sprintf("[%s%s] %.0f%%",
					strings.Repeat("█", filled),
					strings.Repeat("░", barWidth-filled),
					progress*100)
			} else {
				progressBar = "[∞ infinite]"
			}

			// Build the status line
			spinner := spinnerFrames[frameIdx%len(spinnerFrames)]
			frameIdx++

			// Color codes
			green := "\033[32m"
			red := "\033[31m"
			yellow := "\033[33m"
			cyan := "\033[36m"
			reset := "\033[0m"

			// Error indicator
			errorStr := fmt.Sprintf("%s%d%s", green, metrics.Failures, reset)
			if metrics.Failures > 0 {
				errorStr = fmt.Sprintf("%s%d%s", red, metrics.Failures, reset)
			}

			// RPS color (green if close to target, yellow/red if far)
			rpsColor := green
			rpsGap := (avgRPS - float64(cfg.RPS)) / float64(cfg.RPS) * 100
			if rpsGap < -10 {
				rpsColor = red
			} else if rpsGap < -5 {
				rpsColor = yellow
			}

			statusLine := fmt.Sprintf("\r%s %s %s | Reqs: %s%d%s | RPS: %s%.1f%s/%d | Avg: %s%s%s | Err: %s",
				spinner,
				formatDuration(elapsed),
				progressBar,
				cyan, metrics.TotalRequests, reset,
				rpsColor, avgRPS, reset, cfg.RPS,
				yellow, formatDuration(avgLatency), reset,
				errorStr,
			)

			// Add instantaneous RPS
			statusLine += fmt.Sprintf(" | Now: %.0f/s", currentRPS)

			metrics.Unlock()

			// Print status line (overwrite previous)
			fmt.Print("\033[K") // Clear to end of line
			fmt.Print(statusLine)
		}
	}
	}()
	return done
}

var attackCmd = &cobra.Command{
	Use:   "attack",
	Short: "Start a load test against a target URL",
	Long: `Start a load test against a target URL with configurable rate limiting,
workers, and duration. The command sends HTTP requests at the specified rate
and collects detailed metrics including latency percentiles, status codes,
and throughput.

Examples:
  # Basic attack with default settings
  tsunami attack --url https://example.com

  # Attack with custom rate and duration
  tsunami attack --url https://example.com --rate 100/1s --duration 30s

  # Attack with custom headers and save results to JSON
  tsunami attack --url https://api.example.com --headers "Authorization: Bearer token" --output results.json`,
	PersistentPreRun: attackPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := GetAttackConfig(cmd)
		if err != nil {
			fmt.Printf("Error during configuration setup: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Starting attack on %s with %d workers for %v...\n",
			cfg.URL, cfg.Workers, cfg.Duration)
		fmt.Printf("Rate limit: %d requests per second (RPS)\n", cfg.RPS)

		if cfg.Duration == 0 {
			fmt.Println("Duration is 0. Attack running indefinitely (Ctrl+C to stop).")
		}
		fmt.Println()

		stopCh := make(chan struct{})
		progressStopCh := make(chan struct{})

		var progressOnce sync.Once
		stopProgress := func() { progressOnce.Do(func() { close(progressStopCh) }) }

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// Create metrics before attack so we can display live progress
		metrics := attack.NewGlobalMetrics()
		startTime := time.Now()

		// Start live progress display
		progressDone := liveProgress(metrics, cfg, startTime, progressStopCh)

		go func() {
			<-sigCh
			stopProgress()
			fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
			close(stopCh)
		}()

		metrics, elapsedTime, err := attack.RunAttackWithMetrics(cfg, stopCh, metrics)

		// Stop progress display and wait for the goroutine to exit
		stopProgress()
		<-progressDone

		if err != nil {
			fmt.Printf("Error during attack: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nAttack finished. Reporting results...")

		if metrics.TotalRequests > 0 {
			reqPerSec := float64(metrics.TotalRequests) / elapsedTime.Seconds()

			if cfg.Output != "" && cfg.Output != "stdout" {
				jsonData, err := metrics.ToJSON(cfg, elapsedTime, reqPerSec)
				if err != nil {
					fmt.Printf("Error generating JSON report: %v\n", err)
					os.Exit(1)
				}

				err = os.WriteFile(cfg.Output, jsonData, 0644)
				if err != nil {
					fmt.Printf("Error writing output file: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Results saved to %s\n", cfg.Output)
			} else {
				fmt.Printf("\n=== Summary ===\n")
				fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
				fmt.Printf("Successful Requests: %d\n", metrics.Successes)
				fmt.Printf("Failed Requests: %d\n", metrics.Failures)

				fmt.Printf("\n=== Timing ===\n")
				fmt.Printf("Total Elapsed Time: %v\n", elapsedTime.Round(time.Second))

				avgLatency := metrics.TotalLatency / time.Duration(metrics.TotalRequests)
				fmt.Printf("Average Latency: %v\n", avgLatency.Round(time.Millisecond))

				// Min/Max latency
				if metrics.MinLatency < time.Duration(1<<63-1) {
					fmt.Printf("Min Latency: %v\n", metrics.MinLatency.Round(time.Millisecond))
				}
				fmt.Printf("Max Latency: %v\n", metrics.MaxLatency.Round(time.Millisecond))

				fmt.Printf("\n=== Throughput ===\n")
				fmt.Printf("Target RPS: %d\n", cfg.RPS)
				fmt.Printf("Actual RPS: %.2f\n", reqPerSec)
				rpsGap := reqPerSec - float64(cfg.RPS)
				gapPercent := (rpsGap / float64(cfg.RPS)) * 100
				fmt.Printf("RPS Gap: %.2f (%.1f%%)\n", rpsGap, gapPercent)

				fmt.Printf("\n=== Data Transfer ===\n")
				fmt.Printf("Bytes Sent: %s\n", formatBytes(metrics.BytesSent))
				fmt.Printf("Bytes Received: %s\n", formatBytes(metrics.BytesReceived))
				totalBytes := metrics.BytesSent + metrics.BytesReceived
				bandwidth := float64(totalBytes) / elapsedTime.Seconds()
				fmt.Printf("Bandwidth: %s/s\n", formatBytes(uint64(bandwidth)))

				fmt.Printf("\n=== Latency Percentiles ===\n")
				p50, p90, p95, p99 := attack.CalculateAllPercentiles(metrics.Latencies)
				fmt.Printf("  P50: %v\n", p50.Round(time.Millisecond))
				fmt.Printf("  P90: %v\n", p90.Round(time.Millisecond))
				fmt.Printf("  P95: %v\n", p95.Round(time.Millisecond))
				fmt.Printf("  P99: %v\n", p99.Round(time.Millisecond))

				fmt.Printf("\n=== Status Codes ===\n")
				for code, count := range metrics.StatusCodes {
					fmt.Printf("  [%d]: %d (%.2f%%)\n", code, count, float64(count)/float64(metrics.TotalRequests)*100)
				}

				// Error breakdown if there are failures
				if metrics.Failures > 0 {
					fmt.Printf("\n=== Error Breakdown ===\n")
					if count := metrics.ErrorTypes[attack.ErrorTypeTimeout]; count > 0 {
						fmt.Printf("  Timeout: %d\n", count)
					}
					if count := metrics.ErrorTypes[attack.ErrorTypeConnectionRefused]; count > 0 {
						fmt.Printf("  Connection Refused: %d\n", count)
					}
					if count := metrics.ErrorTypes[attack.ErrorTypeDNS]; count > 0 {
						fmt.Printf("  DNS Error: %d\n", count)
					}
					if count := metrics.ErrorTypes[attack.ErrorTypeTLS]; count > 0 {
						fmt.Printf("  TLS Error: %d\n", count)
					}
					if count := metrics.ErrorTypes[attack.ErrorTypeOther]; count > 0 {
						fmt.Printf("  Other: %d\n", count)
					}
				}
			}
		} else {
			fmt.Println("No requests were sent.")
		}
	},
}

// GetAttackConfig builds an AttackConfig from command flags
func GetAttackConfig(cmd *cobra.Command) (*attack.AttackConfig, error) {
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

	rps, err := attack.ParseRateToRPS(rateStr)
	if err != nil {
		return nil, err
	}

	return &attack.AttackConfig{
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

func init() {
	rootCmd.AddCommand(attackCmd)

	attackCmd.Flags().StringP("url", "u", "", "URL to attack")
	attackCmd.MarkFlagRequired("url")

	attackCmd.Flags().StringP("method", "m", defaultMethod, "HTTP method to use (default GET)")
	attackCmd.Flags().StringP("body", "b", defaultBody, "HTTP body to use")
	attackCmd.Flags().StringArrayP("headers", "H", defaultHeaders, "HTTP headers to use")

	attackCmd.Flags().StringP("rate", "r", defaultRate, "Requests per time unit (default 100/1s)")

	attackCmd.Flags().StringP("output", "o", defaultOutput, "Output file path for JSON results (default: stdout for text output)")

	var duration time.Duration
	attackCmd.Flags().DurationVarP(&duration, "duration", "d", defaultDuration, "Duration of the attack (default 0)")

	var timeoutDuration time.Duration
	attackCmd.Flags().DurationVarP(&timeoutDuration, "timeout", "t", defaultTimeoutDuration, "Requests timeout (default 10s)")

	attackCmd.Flags().UintP("workers", "w", defaultWorkers, "Number of workers to use (default 15)")
	attackCmd.Flags().Uint("max-workers", defaultMaxWorkers, "Maximum number of workers to use (default 18446744073709551615)")

	attackCmd.Flags().UintP("connections", "c", defaultConnections, "Number of connections to use (default 100)")
	attackCmd.Flags().Uint("max-connections", defaultMaxConnections, "Maximum number of connections to use (default 18446744073709551615)")

}
