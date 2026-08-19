package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// scrollOffset recomputes a window offset so that cursor stays visible
// within a viewport of the given height over total items, adjusting the
// previous offset as little as possible. It is a pure function, easy to
// unit test independently of Bubbletea.
func scrollOffset(prevOffset, cursor, total, height int) int {
	if height <= 0 || total <= height {
		return 0
	}
	offset := prevOffset
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+height {
		offset = cursor - height + 1
	}
	if offset > total-height {
		offset = total - height
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

// renderList renders the visible window [offset, offset+height) of the
// filtered entries, highlighting the cursor row.
func renderList(entries []domain.LogEntry, indices []int, cursor, offset, height, width int) string {
	if height <= 0 {
		return ""
	}
	var lines []string
	end := offset + height
	if end > len(indices) {
		end = len(indices)
	}
	for i := offset; i < end; i++ {
		e := entries[indices[i]]
		line := formatEntryLine(e, width)
		if i == cursor {
			line = styleSelectedLine.Width(width).Render(trimANSI(line, width))
		}
		lines = append(lines, line)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// formatEntryLine renders a single synthetic-view row: time, severity,
// container name and message, truncated to width.
func formatEntryLine(e domain.LogEntry, width int) string {
	ts := e.Timestamp.Format("15:04:05")
	level := severityStyle(e.Level).Render(padRight(levelBadge(e.Level), 5))
	container := padRight(truncate(e.ContainerName, 16), 16)
	message := strings.ReplaceAll(e.Message, "\n", " ⏎ ")

	prefix := ts + " " + level + " " + container + " "
	visiblePrefixLen := len(ts) + 1 + 5 + 1 + 16 + 1
	remaining := width - visiblePrefixLen
	if remaining < 0 {
		remaining = 0
	}
	return prefix + truncate(message, remaining)
}

// levelBadge returns the short, fixed-width label used for a severity in
// the compact list view; unlike Severity.String(), "UNKNOWN" is shortened
// so it does not get truncated in the 5-column badge.
func levelBadge(s domain.Severity) string {
	if s == domain.SeverityUnknown {
		return "-"
	}
	return s.String()
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

// trimANSI strips lipgloss styling from an already-rendered line before
// re-wrapping it in the selection style, avoiding nested escape sequences.
func trimANSI(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(stripANSI(s))
}

func stripANSI(s string) string {
	var sb strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
