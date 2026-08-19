package services

import (
	"testing"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

func TestHeatmapBuilderBucketsBySeverity(t *testing.T) {
	h := NewHeatmapBuilder(time.Minute)
	base := time.Date(2024, 1, 2, 15, 4, 0, 0, time.UTC)

	h.Add(domain.LogEntry{Timestamp: base, Level: domain.SeverityInfo})
	h.Add(domain.LogEntry{Timestamp: base.Add(10 * time.Second), Level: domain.SeverityError})
	h.Add(domain.LogEntry{Timestamp: base.Add(20 * time.Second), Level: domain.SeverityWarn})
	h.Add(domain.LogEntry{Timestamp: base.Add(90 * time.Second), Level: domain.SeverityInfo})

	buckets := h.Buckets()
	if len(buckets) != 2 {
		t.Fatalf("got %d buckets, want 2", len(buckets))
	}
	first := buckets[0]
	if first.Total != 3 || first.Errors != 1 || first.Warns != 1 {
		t.Errorf("first bucket = %+v, want Total=3 Errors=1 Warns=1", first)
	}
	second := buckets[1]
	if second.Total != 1 {
		t.Errorf("second bucket = %+v, want Total=1", second)
	}
	if !second.Start.After(first.Start) {
		t.Errorf("expected buckets sorted chronologically")
	}
}

func TestHeatmapBuilderDefaultsBucketSize(t *testing.T) {
	h := NewHeatmapBuilder(0)
	if h.bucketSize != time.Minute {
		t.Errorf("bucketSize = %v, want 1m", h.bucketSize)
	}
}

func TestHeatmapBuilderReset(t *testing.T) {
	h := NewHeatmapBuilder(time.Minute)
	h.Add(domain.LogEntry{Timestamp: time.Now()})
	if len(h.Buckets()) == 0 {
		t.Fatal("expected at least one bucket before reset")
	}
	h.Reset()
	if len(h.Buckets()) != 0 {
		t.Fatal("expected no buckets after reset")
	}
}

func TestHeatmapBuilderLast(t *testing.T) {
	h := NewHeatmapBuilder(time.Minute)
	base := time.Date(2024, 1, 2, 15, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		h.Add(domain.LogEntry{Timestamp: base.Add(time.Duration(i) * time.Minute)})
	}

	last := h.Last(2)
	if len(last) != 2 {
		t.Fatalf("got %d buckets, want 2", len(last))
	}
	if !last[len(last)-1].Start.Equal(base.Add(4 * time.Minute)) {
		t.Errorf("last bucket start = %v, want %v", last[len(last)-1].Start, base.Add(4*time.Minute))
	}

	all := h.Last(100)
	if len(all) != 5 {
		t.Fatalf("got %d buckets, want 5 when n exceeds count", len(all))
	}
}
