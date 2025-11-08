package cmd

import (
	"sync"
	"time"
)

type GlobalMetrics struct {
	TotalRequests uint64
	Successes     uint64
	Failures      uint64
	TotalLatency  time.Duration

	Latencies   []time.Duration
	StatusCodes map[int]uint64

	sync.Mutex
}

func NewGlobalMetrics() *GlobalMetrics {
	return &GlobalMetrics{
		StatusCodes: make(map[int]uint64),
	}
}
