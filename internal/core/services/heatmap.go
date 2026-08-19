package services

import (
	"sort"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// HeatmapBuilder aggregates log entries into fixed-size time buckets
// (one minute by default) to feed the density heatmap panel.
type HeatmapBuilder struct {
	bucketSize time.Duration
	buckets    map[int64]*domain.HeatmapBucket
}

// NewHeatmapBuilder creates a builder bucketing entries by bucketSize. A
// zero duration defaults to one minute.
func NewHeatmapBuilder(bucketSize time.Duration) *HeatmapBuilder {
	if bucketSize <= 0 {
		bucketSize = time.Minute
	}
	return &HeatmapBuilder{
		bucketSize: bucketSize,
		buckets:    make(map[int64]*domain.HeatmapBucket),
	}
}

// Add folds a log entry into its time bucket.
func (h *HeatmapBuilder) Add(e domain.LogEntry) {
	key := e.Timestamp.Truncate(h.bucketSize).Unix()
	b, ok := h.buckets[key]
	if !ok {
		b = &domain.HeatmapBucket{Start: e.Timestamp.Truncate(h.bucketSize)}
		h.buckets[key] = b
	}
	b.Total++
	switch e.Level {
	case domain.SeverityError:
		b.Errors++
	case domain.SeverityWarn:
		b.Warns++
	}
}

// Reset clears all accumulated buckets.
func (h *HeatmapBuilder) Reset() {
	h.buckets = make(map[int64]*domain.HeatmapBucket)
}

// Buckets returns the accumulated buckets sorted chronologically.
func (h *HeatmapBuilder) Buckets() []domain.HeatmapBucket {
	out := make([]domain.HeatmapBucket, 0, len(h.buckets))
	for _, b := range h.buckets {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// Last returns, at most, the n most recent buckets in chronological order.
func (h *HeatmapBuilder) Last(n int) []domain.HeatmapBucket {
	all := h.Buckets()
	if len(all) <= n {
		return all
	}
	return all[len(all)-n:]
}
