package domain

import "testing"

func TestHeatmapBucketDensity(t *testing.T) {
	cases := []struct {
		total, max int
		want       float64
	}{
		{total: 0, max: 0, want: 0},
		{total: 5, max: 0, want: 0},
		{total: 5, max: 10, want: 0.5},
		{total: 10, max: 10, want: 1},
		{total: 20, max: 10, want: 1},
	}
	for _, c := range cases {
		b := HeatmapBucket{Total: c.total}
		if got := b.Density(c.max); got != c.want {
			t.Errorf("Density(total=%d, max=%d) = %v, want %v", c.total, c.max, got, c.want)
		}
	}
}
