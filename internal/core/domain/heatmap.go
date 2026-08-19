package domain

import "time"

// HeatmapBucket aggregates counts for every severity within a fixed time
// window (one minute by default).
type HeatmapBucket struct {
	Start  time.Time
	Total  int
	Errors int
	Warns  int
}

// Density returns a value in [0,1] relative to maxTotal, used to pick the
// ASCII/ANSI shading level of the bucket in the TUI.
func (b HeatmapBucket) Density(maxTotal int) float64 {
	if maxTotal <= 0 {
		return 0
	}
	d := float64(b.Total) / float64(maxTotal)
	if d > 1 {
		d = 1
	}
	return d
}
