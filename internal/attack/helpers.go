/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"math"
	"slices"
	"time"
)

// CalculatePercentile calculates the percentile from latencies slice.
func CalculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	slices.Sort(sorted)

	return sorted[percentileIndex(sorted, percentile)]
}

// CalculateAllPercentiles computes P50, P90, P95, P99 with a single sort.
func CalculateAllPercentiles(latencies []time.Duration) (p50, p90, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	slices.Sort(sorted)

	p50 = sorted[percentileIndex(sorted, 50)]
	p90 = sorted[percentileIndex(sorted, 90)]
	p95 = sorted[percentileIndex(sorted, 95)]
	p99 = sorted[percentileIndex(sorted, 99)]
	return
}

func percentileIndex(sorted []time.Duration, p int) int {
	return max(int(math.Ceil(float64(p)/100.0*float64(len(sorted))))-1, 0)
}
