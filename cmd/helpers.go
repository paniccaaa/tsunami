package cmd

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

func globalPreRun(cmd *cobra.Command, args []string) {
	validateAndSetCPUs(cmd)
}

func validateAndSetCPUs(cmd *cobra.Command) {
	rootCmd := cmd.Root()
	cpus, err := rootCmd.PersistentFlags().GetInt("cpus")
	if err != nil {
		fmt.Printf("failed to get cpus: %s\n", err.Error())
		os.Exit(1)
	}
	if cpus <= 0 {
		fmt.Printf("cpus must be a positive integer, got: %d\n", cpus)
		os.Exit(1)
	}
	runtime.GOMAXPROCS(cpus)
}

func attackPreRun(cmd *cobra.Command, args []string) {
	validateRateFormat(cmd)
	validateWorkersAndConnections(cmd)
	validateURL(cmd)
	validateTimeFlags(cmd)
}

var rateRegex = regexp.MustCompile(`^(\d+)/(\d+)(ms|s|m|h)$`)

func validateRateFormat(cmd *cobra.Command) {
	rate, err := cmd.Flags().GetString("rate")
	if err != nil {
		fmt.Printf("failed to get rate: %s\n", err.Error())
		os.Exit(1)
	}

	if !rateRegex.MatchString(rate) {
		fmt.Printf("invalid format rate: %s\n", rate)
		fmt.Println("Expected format: NUMBER/TIME, e.g., 100/1s or 50/1m")
		os.Exit(1)
	}

	matches := rateRegex.FindStringSubmatch(rate)
	rateValue, _ := strconv.Atoi(matches[1])

	if rateValue == 0 {
		fmt.Printf("The number of requests in rate must be greater than 0, got: %s\n", rate)
		os.Exit(1)
	}
}

func validateWorkersAndConnections(cmd *cobra.Command) {
	workers, _ := cmd.Flags().GetUint("workers")
	maxWorkers, _ := cmd.Flags().GetUint("max-workers")

	if workers == 0 {
		fmt.Println("Value for workers must be greater than 0.")
		os.Exit(1)
	}
	if workers > maxWorkers {
		fmt.Printf("Workers count (%d) cannot exceed max-workers (%d)\n", workers, maxWorkers)
		os.Exit(1)
	}

	connections, _ := cmd.Flags().GetUint("connections")
	maxConnections, _ := cmd.Flags().GetUint("max-connections")

	if connections == 0 {
		fmt.Println("Value for connections must be greater than 0.")
		os.Exit(1)
	}
	if connections > maxConnections {
		fmt.Printf("Connections count (%d) cannot exceed max-connections (%d)\n", connections, maxConnections)
		os.Exit(1)
	}
}

func validateURL(cmd *cobra.Command) {
	urlStr, err := cmd.Flags().GetString("url")
	if err != nil {
		fmt.Printf("failed to get url: %s\n", err.Error())
		os.Exit(1)
	}

	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		fmt.Printf("Invalid URL format: %s. Error: %s\n", urlStr, err.Error())
		os.Exit(1)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		fmt.Printf("URL must have a valid scheme (http or https), got: %s\n", u.Scheme)
		os.Exit(1)
	}
}

func validateTimeFlags(cmd *cobra.Command) {
	duration, _ := cmd.Flags().GetDuration("duration")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	if duration < 0 {
		fmt.Println("Duration cannot be negative.")
		os.Exit(1)
	}

	if timeout <= 0 {
		fmt.Println("Timeout must be greater than 0.")
		os.Exit(1)
	}
}

func calculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(math.Ceil(float64(percentile)/100.0*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}

	return sorted[index]
}
func parseRateToRPS(rate string) (rps int, err error) {
	matches := rateRegex.FindStringSubmatch(rate)
	if len(matches) != 4 {
		return 0, fmt.Errorf("invalid rate format: %s. Expected format: N/Vunit (e.g., 100/1s)", rate)
	}

	requests, _ := strconv.Atoi(matches[1])
	value, _ := strconv.Atoi(matches[2])
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

	rpsFloat := float64(requests) / interval.Seconds()

	if rpsFloat > 0 && rpsFloat < 1 {
		rps = 1
	} else {
		rps = int(rpsFloat)
	}

	if rps == 0 {
		rps = 1
	}

	return rps, nil
}
