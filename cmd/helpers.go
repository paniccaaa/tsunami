package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

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
