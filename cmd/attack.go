/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package cmd

import (
	"fmt"
	"math"
	"os"
	"os/signal"
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

		stopCh := make(chan struct{})

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println("\nReceived interrupt signal. Shutting down gracefully...")
			close(stopCh)
		}()

		metrics, elapsedTime, err := attack.RunAttack(cfg, stopCh)
		if err != nil {
			fmt.Printf("Error during attack: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Attack finished. Reporting results...")

		if metrics.TotalRequests > 0 {
			reqPerSec := float64(metrics.TotalRequests) / elapsedTime.Seconds()

			if cfg.Output != "" && cfg.Output != "stdout" {
				jsonData, err := metrics.ToJSON(cfg, elapsedTime, reqPerSec, attack.CalculatePercentile)
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
				fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
				fmt.Printf("Successful Requests: %d\n", metrics.Successes)
				fmt.Printf("Failed Requests: %d\n", metrics.Failures)

				avgLatency := metrics.TotalLatency / time.Duration(metrics.TotalRequests)

				fmt.Printf("Total Elapsed Time: %v\n", elapsedTime.Round(time.Second))
				fmt.Printf("Average Latency: %v\n", avgLatency.Round(time.Millisecond))
				fmt.Printf("Total Throughput (Req/sec): %.2f\n", reqPerSec)

				fmt.Printf("\nLatency Percentiles\n")
				fmt.Printf("  P50: %v\n", attack.CalculatePercentile(metrics.Latencies, 50).Round(time.Millisecond))
				fmt.Printf("  P90: %v\n", attack.CalculatePercentile(metrics.Latencies, 90).Round(time.Millisecond))
				fmt.Printf("  P95: %v\n", attack.CalculatePercentile(metrics.Latencies, 95).Round(time.Millisecond))
				fmt.Printf("  P99: %v\n", attack.CalculatePercentile(metrics.Latencies, 99).Round(time.Millisecond))

				fmt.Printf("\nStatus Codes\n")
				for code, count := range metrics.StatusCodes {
					fmt.Printf("  [%d]: %d (%.2f%%)\n", code, count, float64(count)/float64(metrics.TotalRequests)*100)
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

	rps, err := parseRateToRPS(rateStr)
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
