package cmd

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"

	"github.com/paniccaaa/tsunami/internal/attack"
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
	validateAndSetCPUs(cmd)
	validateRateFormat(cmd)
	validateWorkersAndConnections(cmd)
	validateURL(cmd)
	validateTimeFlags(cmd)
}

func validateRateFormat(cmd *cobra.Command) {
	rate, err := cmd.Flags().GetString("rate")
	if err != nil {
		fmt.Printf("failed to get rate: %s\n", err.Error())
		os.Exit(1)
	}

	if !attack.RateRegex.MatchString(rate) {
		fmt.Printf("invalid format rate: %s\n", rate)
		fmt.Println("Expected format: NUMBER/TIME, e.g., 100/1s or 50/1m")
		os.Exit(1)
	}

	matches := attack.RateRegex.FindStringSubmatch(rate)
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

