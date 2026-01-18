/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"time"
)

// AttackConfig holds the configuration for a load test
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

// RequestResult represents the result of a single HTTP request
type RequestResult struct {
	StatusCode int
	Latency    time.Duration
	Success    bool
}
