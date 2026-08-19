package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// renderDetail renders the Kibana-style expanded, field-by-field view of a
// single log entry: its metadata first, then every parsed JSON field
// sorted by key, then the raw message.
func renderDetail(e domain.LogEntry, width int) string {
	var lines []string

	addField := func(key, value string) {
		lines = append(lines, styleFieldKey.Render(padRight(key, 14))+value)
	}

	addField("timestamp", e.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"))
	addField("container", e.ContainerName)
	addField("stream", string(e.Stream))
	addField("level", severityStyle(e.Level).Render(e.Level.String()))
	addField("format", format(e))

	if len(e.Fields) > 0 {
		lines = append(lines, "", styleFieldKey.Render("fields"))
		keys := make([]string, 0, len(e.Fields))
		for k := range e.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			addField("  "+k, fmt.Sprintf("%v", e.Fields[k]))
		}
	}

	lines = append(lines, "", styleFieldKey.Render("message"))
	lines = append(lines, wrap(e.Message, width)...)

	return strings.Join(lines, "\n")
}

func format(e domain.LogEntry) string {
	switch {
	case e.IsJSON:
		return "json"
	case e.Multiline:
		return "multiline"
	default:
		return "text"
	}
}

// wrap breaks s into lines of at most width runes, preserving existing
// newlines (e.g. inside a stack trace).
func wrap(s string, width int) []string {
	if width <= 0 {
		width = 80
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		runes := []rune(para)
		if len(runes) == 0 {
			out = append(out, "")
			continue
		}
		for len(runes) > 0 {
			n := width
			if n > len(runes) {
				n = len(runes)
			}
			out = append(out, string(runes[:n]))
			runes = runes[n:]
		}
	}
	return out
}
