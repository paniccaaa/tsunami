/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"math"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultDuration        = 0
	defaultTimeoutDuration = time.Second * 10
	defaultRate            = "100/1s"
	defaultWorkers         = 15
	defaultMaxWorkers      = math.MaxUint
	defaultConnections     = 100
	defaultMaxConnections  = math.MaxUint
	defaultMethod          = "GET"
	defaultOutput          = "stdout"
	defaultBody            = ""
)

var defaultHeaders = []string{}

var attackCmd = &cobra.Command{
	Use:   "attack",
	Short: "Attack a target",
	Long:  "Attack a target",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("attack called")
	},
}

func init() {
	rootCmd.AddCommand(attackCmd)

	attackCmd.Flags().StringP("url", "u", "", "URL to attack")
	attackCmd.MarkFlagRequired("url")

	attackCmd.Flags().StringP("method", "m", defaultMethod, "HTTP method to use (default GET)")
	attackCmd.Flags().StringP("body", "b", defaultBody, "HTTP body to use")
	attackCmd.Flags().StringArrayP("headers", "h", defaultHeaders, "HTTP headers to use")

	attackCmd.Flags().StringP("rate", "r", defaultRate, "Requests per time unit (default 100/1s)")

	attackCmd.Flags().StringP("output", "o", defaultOutput, "Output file (default stdout)")

	var duration time.Duration
	attackCmd.Flags().DurationVarP(&duration, "duration", "d", defaultDuration, "Duration of the attack (default 0)")

	var timeoutDuration time.Duration
	attackCmd.Flags().DurationVarP(&timeoutDuration, "timeout", "t", defaultTimeoutDuration, "Requests timeout (default 10s)")

	attackCmd.Flags().UintP("workers", "w", defaultWorkers, "Number of workers to use (default 15)")
	attackCmd.Flags().UintP("max-workers", "mw", defaultMaxWorkers, "Maximum number of workers to use (default 18446744073709551615)")

	attackCmd.Flags().UintP("connections", "c", defaultConnections, "Number of connections to use (default 100)")
	attackCmd.Flags().UintP("max-connections", "mc", defaultMaxConnections, "Maximum number of connections to use (default 18446744073709551615)")

}
