package attack

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// RateRegex matches rate strings in the format N/Vunit (e.g. 100/1s, 50/1m).
var RateRegex = regexp.MustCompile(`^(\d+)/(\d+)(ms|s|m|h)$`)

// ParseRateToRPS converts a rate string (e.g. "100/1s") to requests per second.
// Sub-1-RPS values are floored to 1.
func ParseRateToRPS(rate string) (int, error) {
	matches := RateRegex.FindStringSubmatch(rate)
	if len(matches) != 4 {
		return 0, fmt.Errorf("invalid rate format: %s. Expected format: N/Vunit (e.g., 100/1s)", rate)
	}

	requests, _ := strconv.Atoi(matches[1])
	value, _ := strconv.Atoi(matches[2])
	unit := matches[3]

	var interval time.Duration
	switch unit {
	case "ms":
		interval = time.Millisecond * time.Duration(value)
	case "s":
		interval = time.Second * time.Duration(value)
	case "m":
		interval = time.Minute * time.Duration(value)
	case "h":
		interval = time.Hour * time.Duration(value)
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}

	rpsFloat := float64(requests) / interval.Seconds()
	if rpsFloat < 1 {
		return 1, nil
	}
	return int(rpsFloat), nil
}
