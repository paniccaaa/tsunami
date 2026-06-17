package attack

import (
	"time"
)

type AttackConfig struct {
	URL         string
	Method      string
	Body        string
	Headers     []string
	Output      string
	Duration    time.Duration
	Timeout     time.Duration
	Workers     uint
	Connections uint

	RPS int
}

type RequestResult struct {
	StatusCode    int
	Latency       time.Duration
	Success       bool
	ErrorType     ErrorType
	BytesSent     uint64
	BytesReceived uint64
}
