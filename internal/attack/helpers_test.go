/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package attack

import (
	"testing"
	"time"
)

func TestCalculatePercentile(t *testing.T) {
	type args struct {
		latencies  []time.Duration
		percentile int
	}
	ms := time.Millisecond
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "empty slice returns zero",
			args: args{latencies: []time.Duration{}, percentile: 50},
			want: 0,
		},
		{
			name: "single element any percentile",
			args: args{latencies: []time.Duration{100 * ms}, percentile: 50},
			want: 100 * ms,
		},
		{
			name: "p50 odd-length slice",
			args: args{latencies: []time.Duration{1 * ms, 2 * ms, 3 * ms, 4 * ms, 5 * ms}, percentile: 50},
			want: 3 * ms,
		},
		{
			name: "p50 even-length slice",
			args: args{latencies: []time.Duration{10 * ms, 20 * ms, 30 * ms, 40 * ms}, percentile: 50},
			want: 20 * ms,
		},
		{
			name: "p75",
			args: args{latencies: []time.Duration{10 * ms, 20 * ms, 30 * ms, 40 * ms}, percentile: 75},
			want: 30 * ms,
		},
		{
			name: "p100 returns max",
			args: args{latencies: []time.Duration{1 * ms, 2 * ms, 3 * ms, 4 * ms, 5 * ms}, percentile: 100},
			want: 5 * ms,
		},
		{
			name: "p0 returns min",
			args: args{latencies: []time.Duration{1 * ms, 2 * ms, 3 * ms, 4 * ms, 5 * ms}, percentile: 0},
			want: 1 * ms,
		},
		{
			name: "unsorted input is sorted before calculation",
			args: args{latencies: []time.Duration{50 * ms, 10 * ms, 30 * ms, 20 * ms, 40 * ms}, percentile: 50},
			want: 30 * ms,
		},
		{
			name: "p99 on 100 elements",
			args: args{
				latencies: func() []time.Duration {
					d := make([]time.Duration, 100)
					for i := range d {
						d[i] = time.Duration(i+1) * ms
					}
					return d
				}(),
				percentile: 99,
			},
			want: 99 * ms,
		},
		{
			name: "p95 on 100 elements",
			args: args{
				latencies: func() []time.Duration {
					d := make([]time.Duration, 100)
					for i := range d {
						d[i] = time.Duration(i+1) * ms
					}
					return d
				}(),
				percentile: 95,
			},
			want: 95 * ms,
		},
		{
			name: "p90 on 10 elements",
			args: args{latencies: []time.Duration{10 * ms, 20 * ms, 30 * ms, 40 * ms, 50 * ms, 60 * ms, 70 * ms, 80 * ms, 90 * ms, 100 * ms}, percentile: 90},
			want: 90 * ms,
		},
		{
			name: "duplicate values",
			args: args{latencies: []time.Duration{100 * ms, 100 * ms, 200 * ms, 200 * ms}, percentile: 50},
			want: 100 * ms,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculatePercentile(tt.args.latencies, tt.args.percentile); got != tt.want {
				t.Errorf("CalculatePercentile() = %v, want %v", got, tt.want)
			}
		})
	}
}
