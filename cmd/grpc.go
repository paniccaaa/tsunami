package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
	"github.com/paniccaaa/tsunami/internal/grpcattack"
	"github.com/spf13/cobra"
)

// gRPC status code names for human-readable output.
var grpcStatusNames = map[int]string{
	0: "OK", 1: "CANCELLED", 2: "UNKNOWN", 3: "INVALID_ARGUMENT",
	4: "DEADLINE_EXCEEDED", 5: "NOT_FOUND", 6: "ALREADY_EXISTS",
	7: "PERMISSION_DENIED", 8: "RESOURCE_EXHAUSTED", 9: "FAILED_PRECONDITION",
	10: "ABORTED", 11: "OUT_OF_RANGE", 12: "UNIMPLEMENTED", 13: "INTERNAL",
	14: "UNAVAILABLE", 15: "DATA_LOSS", 16: "UNAUTHENTICATED",
}

var grpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "Start a gRPC load test against a target server",
	Long: `Start a gRPC unary load test against a target gRPC server.

Service and method are resolved via server reflection by default.
Provide --proto to use a local .proto file instead.

Examples:
  # Unary call using server reflection
  tsunami grpc --target localhost:50051 \
    --service helloworld.Greeter --method SayHello \
    --data '{"name":"world"}' --insecure --duration 30s

  # Using a .proto file
  tsunami grpc --target api.example.com:443 \
    --proto ./service.proto \
    --service myapp.UserService --method GetUser \
    --data '{"id":"123"}' --rate 200/1s`,
	PersistentPreRun: grpcPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := buildGRPCConfig(cmd)
		if err != nil {
			fmt.Printf("configuration error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Starting gRPC attack on %s (%s/%s) with %d workers...\n",
			cfg.Target, cfg.Service, cfg.Method, cfg.Workers)
		fmt.Printf("Rate limit: %d RPS\n", cfg.RPS)
		if cfg.Duration == 0 {
			fmt.Println("Duration: infinite (Ctrl+C to stop)")
		} else {
			fmt.Printf("Duration: %v\n", cfg.Duration)
		}
		if cfg.ProtoFile != "" {
			fmt.Printf("Proto file: %s\n", cfg.ProtoFile)
		} else {
			fmt.Println("Method resolution: server reflection")
		}
		fmt.Println()

		stopCh := make(chan struct{})
		progressStopCh := make(chan struct{})

		var progressOnce sync.Once
		stopProgress := func() { progressOnce.Do(func() { close(progressStopCh) }) }

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		metrics := attack.NewGlobalMetrics()
		startTime := time.Now()

		progressDone := liveProgress(metrics, &attack.AttackConfig{
			Duration: cfg.Duration,
			RPS:      cfg.RPS,
		}, startTime, progressStopCh)

		go func() {
			<-sigCh
			stopProgress()
			fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
			close(stopCh)
		}()

		metrics, elapsedTime, err := grpcattack.RunAttack(cfg, stopCh, metrics)

		stopProgress()
		<-progressDone

		if err != nil {
			fmt.Printf("Error during gRPC attack: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nAttack finished. Reporting results...")

		if metrics.TotalRequests == 0 {
			fmt.Println("No requests were sent.")
			return
		}

		reqPerSec := float64(metrics.TotalRequests) / elapsedTime.Seconds()

		if cfg.Output != "" && cfg.Output != "stdout" {
			jsonData, err := metrics.ToJSON(&attack.AttackConfig{
				URL:      cfg.Target,
				Duration: cfg.Duration,
				Timeout:  cfg.Timeout,
				Workers:  cfg.Workers,
				RPS:      cfg.RPS,
			}, elapsedTime, reqPerSec)
			if err != nil {
				fmt.Printf("Error generating JSON report: %v\n", err)
				os.Exit(1)
			}
			if err := os.WriteFile(cfg.Output, jsonData, 0644); err != nil {
				fmt.Printf("Error writing output file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Results saved to %s\n", cfg.Output)
			return
		}

		fmt.Printf("\n=== Summary ===\n")
		fmt.Printf("Total Requests:     %d\n", metrics.TotalRequests)
		fmt.Printf("Successful:         %d\n", metrics.Successes)
		fmt.Printf("Failed:             %d\n", metrics.Failures)

		fmt.Printf("\n=== Timing ===\n")
		fmt.Printf("Total Elapsed Time: %v\n", elapsedTime.Round(time.Second))
		avgLatency := metrics.TotalLatency / time.Duration(metrics.TotalRequests)
		fmt.Printf("Average Latency:    %v\n", avgLatency.Round(time.Millisecond))
		if metrics.MinLatency < time.Duration(1<<63-1) {
			fmt.Printf("Min Latency:        %v\n", metrics.MinLatency.Round(time.Millisecond))
		}
		fmt.Printf("Max Latency:        %v\n", metrics.MaxLatency.Round(time.Millisecond))

		fmt.Printf("\n=== Throughput ===\n")
		fmt.Printf("Target RPS:  %d\n", cfg.RPS)
		fmt.Printf("Actual RPS:  %.2f\n", reqPerSec)

		fmt.Printf("\n=== Latency Percentiles ===\n")
		p50, p90, p95, p99 := attack.CalculateAllPercentiles(metrics.Latencies)
		fmt.Printf("  P50: %v\n", p50.Round(time.Millisecond))
		fmt.Printf("  P90: %v\n", p90.Round(time.Millisecond))
		fmt.Printf("  P95: %v\n", p95.Round(time.Millisecond))
		fmt.Printf("  P99: %v\n", p99.Round(time.Millisecond))

		fmt.Printf("\n=== gRPC Status Codes ===\n")
		for code, count := range metrics.StatusCodes {
			name := grpcStatusNames[code]
			if name == "" {
				name = fmt.Sprintf("CODE_%d", code)
			}
			fmt.Printf("  [%s]: %d (%.2f%%)\n", name, count,
				float64(count)/float64(metrics.TotalRequests)*100)
		}

		if metrics.Failures > 0 {
			fmt.Printf("\n=== Error Breakdown ===\n")
			for t, count := range metrics.ErrorTypes {
				if count > 0 {
					fmt.Printf("  %s: %d\n", t, count)
				}
			}
		}
	},
}

func buildGRPCConfig(cmd *cobra.Command) (*grpcattack.Config, error) {
	target, _ := cmd.Flags().GetString("target")
	service, _ := cmd.Flags().GetString("service")
	method, _ := cmd.Flags().GetString("method")
	data, _ := cmd.Flags().GetString("data")
	protoFile, _ := cmd.Flags().GetString("proto")
	metadata, _ := cmd.Flags().GetStringArray("grpc-metadata")
	insecure, _ := cmd.Flags().GetBool("insecure")
	caCert, _ := cmd.Flags().GetString("ca-cert")
	rateStr, _ := cmd.Flags().GetString("rate")
	duration, _ := cmd.Flags().GetDuration("duration")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	workers, _ := cmd.Flags().GetUint("workers")
	connections, _ := cmd.Flags().GetUint("connections")
	output, _ := cmd.Flags().GetString("output")

	rps, err := attack.ParseRateToRPS(rateStr)
	if err != nil {
		return nil, err
	}

	return &grpcattack.Config{
		Target:      target,
		Service:     service,
		Method:      method,
		Data:        data,
		ProtoFile:   protoFile,
		Metadata:    metadata,
		Insecure:    insecure,
		CACert:      caCert,
		Duration:    duration,
		Timeout:     timeout,
		Workers:     workers,
		Connections: connections,
		RPS:         rps,
		Output:      output,
	}, nil
}

func init() {
	rootCmd.AddCommand(grpcCmd)

	grpcCmd.Flags().String("target", "", "gRPC server address in host:port format (required)")
	grpcCmd.MarkFlagRequired("target")

	grpcCmd.Flags().String("service", "", "Fully qualified service name, e.g. helloworld.Greeter (required)")
	grpcCmd.MarkFlagRequired("service")

	grpcCmd.Flags().String("method", "", "RPC method name, e.g. SayHello (required)")
	grpcCmd.MarkFlagRequired("method")

	grpcCmd.Flags().String("data", "{}", "JSON request payload")
	grpcCmd.Flags().String("proto", "", "Path to .proto file (uses server reflection if omitted)")
	grpcCmd.Flags().StringArray("grpc-metadata", []string{}, "gRPC metadata as key:value pairs")
	grpcCmd.Flags().Bool("insecure", false, "Disable TLS (for local testing)")
	grpcCmd.Flags().String("ca-cert", "", "Path to PEM-encoded CA certificate")

	grpcCmd.Flags().StringP("rate", "r", defaultRate, "Requests per time unit (e.g. 100/1s)")
	grpcCmd.Flags().DurationP("duration", "d", defaultDuration, "Test duration (0 = infinite)")
	grpcCmd.Flags().DurationP("timeout", "t", defaultTimeoutDuration, "Per-request timeout")
	grpcCmd.Flags().UintP("workers", "w", defaultWorkers, "Number of concurrent workers")
	grpcCmd.Flags().UintP("connections", "c", 4, "Number of gRPC channels in the connection pool")
	grpcCmd.Flags().StringP("output", "o", defaultOutput, "Output file path for JSON results")
}
