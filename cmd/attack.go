/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// attackCmd represents the attack command
var attackCmd = &cobra.Command{
	Use: "attack",
	// TODO: add desc. for attack command
	Short: "Attack a target",
	Long:  "Attack a target",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("attack called")
	},
}

func init() {
	rootCmd.AddCommand(attackCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// attackCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// attackCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
