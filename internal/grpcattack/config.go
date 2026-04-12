package grpcattack

import "time"

// Config holds all parameters for a gRPC load test.
type Config struct {
	// Target is the gRPC server address in host:port format.
	Target string

	// Service is the fully qualified service name, e.g. "helloworld.Greeter".
	Service string

	// Method is the RPC method name, e.g. "SayHello".
	Method string

	// ProtoFile is an optional path to a .proto source file.
	// When empty, server reflection is used instead.
	ProtoFile string

	// Data is the JSON-encoded request payload. Defaults to "{}".
	Data string

	// Metadata is a list of "key:value" strings sent as gRPC metadata.
	Metadata []string

	// Insecure disables TLS verification (useful for local testing).
	Insecure bool

	// CACert is an optional path to a PEM-encoded CA certificate for TLS.
	CACert string

	Duration    time.Duration
	Timeout     time.Duration
	Workers     uint
	Connections uint // number of gRPC channels in the pool
	RPS         int
	Output      string
}
