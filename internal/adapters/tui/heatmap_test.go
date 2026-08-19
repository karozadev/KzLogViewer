package tui

import (
	"testing"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

func TestRenderHeatmapWidthAndPadding(t *testing.T) {
	buckets := []domain.HeatmapBucket{
		{Start: time.Now(), Total: 5},
		{Start: time.Now(), Total: 10},
	}
	got := renderHeatmap(buckets, 5)
	if len([]rune(stripANSI(got))) != 5 {
		t.Errorf("expected padded output of width 5, got %q (%d runes)", got, len([]rune(stripANSI(got))))
	}
}

func TestRenderHeatmapTruncatesToWidth(t *testing.T) {
	buckets := make([]domain.HeatmapBucket, 10)
	for i := range buckets {
		buckets[i] = domain.HeatmapBucket{Total: i}
	}
	got := stripANSI(renderHeatmap(buckets, 3))
	if len([]rune(got)) != 3 {
		t.Errorf("expected 3 runes, got %d: %q", len([]rune(got)), got)
	}
}

func TestRenderHeatmapZeroWidth(t *testing.T) {
	if got := renderHeatmap(nil, 0); got != "" {
		t.Errorf("expected empty string for zero width, got %q", got)
	}
}

func TestBucketGlyphEmptyBucket(t *testing.T) {
	glyph := bucketGlyph(domain.HeatmapBucket{Total: 0}, 10)
	if glyph != " " {
		t.Errorf("expected space for empty bucket, got %q", glyph)
	}
}

func TestBucketGlyphColorsByErrors(t *testing.T) {
	glyph := stripANSI(bucketGlyph(domain.HeatmapBucket{Total: 5, Errors: 2}, 10))
	if glyph == "" {
		t.Error("expected a non-empty glyph")
	}
}
