/*
Copyright © 2025 NAME HERE <semaadamenko1@gmail.com>
*/
package cmd

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/ratelimit"
)

const (
	defaultDuration        = 0
	defaultTimeoutDuration = time.Second * 10

	defaultRate = "100/1s"

	defaultWorkers    = 15
	defaultMaxWorkers = math.MaxUint

	defaultConnections    = 100
	defaultMaxConnections = math.MaxUint

	defaultMethod = "GET"
	defaultOutput = "stdout"
	defaultBody   = ""
)

var attackCmd = &cobra.Command{
	Use:              "attack",
	Short:            "Attack a target",
	Long:             "Attack a target",
	PersistentPreRun: attackPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := GetAttackConfig(cmd)
		if err != nil {
			fmt.Printf("Error during configuration setup: %v\n", err)
			os.Exit(1)
		}

		client := createHTTPClient(cfg)
		metrics := NewGlobalMetrics()

		jobs := make(chan struct{}, cfg.Workers)
		results := make(chan RequestResult, cfg.Workers*2)
		var wg sync.WaitGroup

		for i := uint(1); i <= cfg.Workers; i++ {
			wg.Add(1)
			go worker(client, cfg, jobs, results, &wg)
		}

		rateLimiter := ratelimit.New(cfg.RPS)

		fmt.Printf("Starting attack on %s with %d workers for %v...\n",
			cfg.URL, cfg.Workers, cfg.Duration)
		fmt.Printf("Rate limit: %d requests per second (RPS)\n", cfg.RPS)

		stopCh := make(chan struct{})
		startTime := time.Now()

		var attackTimer *time.Timer
		if cfg.Duration > 0 {
			attackTimer = time.NewTimer(cfg.Duration)
			go func() {
				<-attackTimer.C
				close(stopCh)
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
					case jobs <- struct{}{}:
					default:
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

		if cfg.Duration == 0 {
			fmt.Println("Duration is 0. Attack running indefinitely (Ctrl+C to stop).")
			select {}
		}

		<-stopCh

		wg.Wait()
		close(results)
		collectorWG.Wait()

		elapsedTime := time.Since(startTime)
		if cfg.Duration > 0 {
			elapsedTime = cfg.Duration
		}

		fmt.Println("Attack finished. Reporting results...")

		if metrics.TotalRequests > 0 {
			reqPerSec := float64(metrics.TotalRequests) / elapsedTime.Seconds()

			fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
			fmt.Printf("Successful Requests: %d\n", metrics.Successes)
			fmt.Printf("Failed Requests: %d\n", metrics.Failures)

			avgLatency := metrics.TotalLatency / time.Duration(metrics.TotalRequests)

			fmt.Printf("Total Elapsed Time: %v\n", elapsedTime.Round(time.Second))
			fmt.Printf("Average Latency: %v\n", avgLatency.Round(time.Millisecond))
			fmt.Printf("Total Throughput (Req/sec): %.2f\n", reqPerSec)

			fmt.Printf("\nLatency Percentiles\n")
			fmt.Printf("  P50: %v\n", calculatePercentile(metrics.Latencies, 50).Round(time.Millisecond))
			fmt.Printf("  P90: %v\n", calculatePercentile(metrics.Latencies, 90).Round(time.Millisecond))
			fmt.Printf("  P95: %v\n", calculatePercentile(metrics.Latencies, 95).Round(time.Millisecond))
			fmt.Printf("  P99: %v\n", calculatePercentile(metrics.Latencies, 99).Round(time.Millisecond))

			fmt.Printf("\nStatus Codes\n")
			for code, count := range metrics.StatusCodes {
				fmt.Printf("  [%d]: %d (%.2f%%)\n", code, count, float64(count)/float64(metrics.TotalRequests)*100)
			}
		} else {
			fmt.Println("No requests were sent.")
		}
	},
}

func init() {
	rootCmd.AddCommand(attackCmd)

	attackCmd.Flags().StringP("url", "u", "", "URL to attack")
	attackCmd.MarkFlagRequired("url")

	attackCmd.Flags().StringP("method", "m", defaultMethod, "HTTP method to use (default GET)")
	attackCmd.Flags().StringP("body", "b", defaultBody, "HTTP body to use")
	attackCmd.Flags().StringArrayP("headers", "H", defaultHeaders, "HTTP headers to use")

	attackCmd.Flags().StringP("rate", "r", defaultRate, "Requests per time unit (default 100/1s)")

	attackCmd.Flags().StringP("output", "o", defaultOutput, "Output file (default stdout)")

	var duration time.Duration
	attackCmd.Flags().DurationVarP(&duration, "duration", "d", defaultDuration, "Duration of the attack (default 0)")

	var timeoutDuration time.Duration
	attackCmd.Flags().DurationVarP(&timeoutDuration, "timeout", "t", defaultTimeoutDuration, "Requests timeout (default 10s)")

	attackCmd.Flags().UintP("workers", "w", defaultWorkers, "Number of workers to use (default 15)")
	attackCmd.Flags().Uint("max-workers", defaultMaxWorkers, "Maximum number of workers to use (default 18446744073709551615)")

	attackCmd.Flags().UintP("connections", "c", defaultConnections, "Number of connections to use (default 100)")
	attackCmd.Flags().Uint("max-connections", defaultMaxConnections, "Maximum number of connections to use (default 18446744073709551615)")

}
