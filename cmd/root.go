/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tsunami",
	Short: "HTTP load testing tool",
	Long: `Tsunami is a modern HTTP load testing tool designed for performance testing
and stress testing of web services. It provides configurable rate limiting,
detailed metrics, and JSON export capabilities.

Use 'tsunami attack' to start a load test against a target URL.
See 'tsunami attack --help' for detailed usage information.`,
	PersistentPreRun: globalPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		version, _ := cmd.Flags().GetBool("version")
		if version {
			fmt.Printf("tsunami version %s\nRuntime: %s %s/%s\n",
				Version,
				runtime.Version(),
				runtime.GOOS,
				runtime.GOARCH,
			)
			os.Exit(0)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Global persistent flags
	rootCmd.PersistentFlags().Int("cpus", runtime.NumCPU(), "Number of CPUs to use")

	// Local root flags
	rootCmd.Flags().BoolP("version", "v", false, "Print version and exit")
}
