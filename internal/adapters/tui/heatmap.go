package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// shades are the ASCII/ANSI density levels used to draw the heatmap strip,
// from empty to saturated.
var shades = []rune(" .:-=+*#%@")

// renderHeatmap draws a one-line-per-metric density strip for the given
// buckets, one column per bucket, shaded by traffic volume and colored by
// the dominant severity of that minute. It is a pure function so it can be
// unit tested without a terminal.
func renderHeatmap(buckets []domain.HeatmapBucket, width int) string {
	if width <= 0 {
		return ""
	}
	if len(buckets) > width {
		buckets = buckets[len(buckets)-width:]
	}

	max := 0
	for _, b := range buckets {
		if b.Total > max {
			max = b.Total
		}
	}

	var sb strings.Builder
	for _, b := range buckets {
		sb.WriteString(bucketGlyph(b, max))
	}
	pad := width - len(buckets)
	if pad > 0 {
		sb.WriteString(strings.Repeat(" ", pad))
	}
	return sb.String()
}

func bucketGlyph(b domain.HeatmapBucket, max int) string {
	density := b.Density(max)
	idx := int(density * float64(len(shades)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(shades) {
		idx = len(shades) - 1
	}
	glyph := string(shades[idx])
	if b.Total == 0 {
		return glyph
	}

	color := colorInfo
	switch {
	case b.Errors > 0:
		color = colorError
	case b.Warns > 0:
		color = colorWarn
	}
	return lipgloss.NewStyle().Foreground(color).Render(glyph)
}
