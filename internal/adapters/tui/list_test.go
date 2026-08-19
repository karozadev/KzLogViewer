package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

func TestScrollOffset(t *testing.T) {
	cases := []struct {
		name                              string
		prevOffset, cursor, total, height int
		want                              int
	}{
		{"fits entirely", 0, 3, 5, 10, 0},
		{"cursor above window scrolls up", 5, 2, 20, 10, 2},
		{"cursor below window scrolls down", 0, 15, 20, 10, 6},
		{"offset clamped to end", 0, 19, 20, 10, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := scrollOffset(c.prevOffset, c.cursor, c.total, c.height)
			if got != c.want {
				t.Errorf("scrollOffset(%d,%d,%d,%d) = %d, want %d", c.prevOffset, c.cursor, c.total, c.height, got, c.want)
			}
		})
	}
}

func TestFormatEntryLineTruncatesToWidth(t *testing.T) {
	e := domain.LogEntry{
		Timestamp:     time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		Level:         domain.SeverityError,
		ContainerName: "web",
		Message:       strings.Repeat("x", 200),
	}
	line := stripANSI(formatEntryLine(e, 40))
	if len([]rune(line)) > 40 {
		t.Errorf("line too long: %d runes", len([]rune(line)))
	}
	if !strings.Contains(line, "15:04:05") {
		t.Errorf("expected timestamp in line: %q", line)
	}
	if !strings.Contains(line, "ERROR") {
		t.Errorf("expected level in line: %q", line)
	}
}

func TestFormatEntryLineReplacesNewlines(t *testing.T) {
	e := domain.LogEntry{Message: "line1\nline2", ContainerName: "web"}
	line := formatEntryLine(e, 80)
	if strings.Contains(line, "\n") {
		t.Errorf("expected no raw newline in single-line rendering: %q", line)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short string changed it: %q", got)
	}
	if got := truncate("hello world", 5); got != "hell…" {
		t.Errorf("truncate = %q, want hell…", got)
	}
	if got := truncate("hello", 0); got != "" {
		t.Errorf("truncate to 0 = %q, want empty", got)
	}
}

func TestPadRight(t *testing.T) {
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight = %q", got)
	}
	if got := padRight("abcdef", 3); got != "abc" {
		t.Errorf("padRight should clip: %q", got)
	}
}
