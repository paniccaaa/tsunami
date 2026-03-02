/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"math"
	"slices"
	"time"
)

// CalculatePercentile calculates the percentile from latencies slice
func CalculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	slices.Sort(sorted)

	index := max(int(math.Ceil(float64(percentile)/100.0*float64(len(sorted))))-1, 0)

	return sorted[index]
}
