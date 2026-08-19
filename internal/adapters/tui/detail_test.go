package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

func TestRenderDetailIncludesFieldsSorted(t *testing.T) {
	e := domain.LogEntry{
		Timestamp:     time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		ContainerName: "web",
		Stream:        domain.StreamStdout,
		Level:         domain.SeverityWarn,
		IsJSON:        true,
		Message:       "disk usage high",
		Fields: map[string]any{
			"zeta":  "last",
			"alpha": "first",
		},
	}
	out := stripANSI(renderDetail(e, 80))

	if !strings.Contains(out, "container     web") {
		t.Errorf("expected container field, got:\n%s", out)
	}
	if !strings.Contains(out, "json") {
		t.Errorf("expected format=json, got:\n%s", out)
	}
	alphaIdx := strings.Index(out, "alpha")
	zetaIdx := strings.Index(out, "zeta")
	if alphaIdx == -1 || zetaIdx == -1 || alphaIdx > zetaIdx {
		t.Errorf("expected fields sorted alphabetically, got:\n%s", out)
	}
}

func TestWrapPreservesNewlinesAndBreaksLongLines(t *testing.T) {
	lines := wrap("short\n"+strings.Repeat("a", 12), 5)
	if lines[0] != "short" {
		t.Errorf("first line = %q, want short", lines[0])
	}
	if len(lines) != 1+3 {
		t.Fatalf("got %d lines, want 4: %v", len(lines), lines)
	}
	for _, l := range lines[1:] {
		if len([]rune(l)) > 5 {
			t.Errorf("line too long: %q", l)
		}
	}
}

func TestFormatClassifiesEntry(t *testing.T) {
	if got := format(domain.LogEntry{IsJSON: true}); got != "json" {
		t.Errorf("got %q, want json", got)
	}
	if got := format(domain.LogEntry{Multiline: true}); got != "multiline" {
		t.Errorf("got %q, want multiline", got)
	}
	if got := format(domain.LogEntry{}); got != "text" {
		t.Errorf("got %q, want text", got)
	}
}
