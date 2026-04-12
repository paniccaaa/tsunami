package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func grpcPreRun(cmd *cobra.Command, args []string) {
	validateAndSetCPUs(cmd)
	validateGRPCTarget(cmd)
	validateGRPCMethod(cmd)
	validateGRPCData(cmd)
	validateGRPCProtoFile(cmd)
	validateGRPCTimeFlags(cmd)
}

func validateGRPCTarget(cmd *cobra.Command) {
	target, _ := cmd.Flags().GetString("target")
	host, port, err := net.SplitHostPort(target)
	if err != nil || host == "" || port == "" {
		fmt.Printf("invalid --target %q: must be in host:port format (e.g. localhost:50051)\n", target)
		os.Exit(1)
	}
}

func validateGRPCMethod(cmd *cobra.Command) {
	service, _ := cmd.Flags().GetString("service")
	method, _ := cmd.Flags().GetString("method")
	if service == "" {
		fmt.Println("--service is required (e.g. helloworld.Greeter)")
		os.Exit(1)
	}
	if method == "" {
		fmt.Println("--method is required (e.g. SayHello)")
		os.Exit(1)
	}
	if strings.Contains(method, ".") {
		fmt.Printf("--method should be a simple name without dots (e.g. SayHello), got: %q\n", method)
		os.Exit(1)
	}
}

func validateGRPCData(cmd *cobra.Command) {
	data, _ := cmd.Flags().GetString("data")
	if data == "" {
		return
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(data), &js); err != nil {
		fmt.Printf("--data is not valid JSON: %v\n", err)
		os.Exit(1)
	}
}

func validateGRPCProtoFile(cmd *cobra.Command) {
	protoFile, _ := cmd.Flags().GetString("proto")
	if protoFile == "" {
		return
	}
	if _, err := os.Stat(protoFile); err != nil {
		fmt.Printf("--proto file not found: %q\n", protoFile)
		os.Exit(1)
	}
}

func validateGRPCTimeFlags(cmd *cobra.Command) {
	timeout, _ := cmd.Flags().GetDuration("timeout")
	if timeout <= 0 {
		fmt.Println("Timeout must be greater than 0.")
		os.Exit(1)
	}
}
