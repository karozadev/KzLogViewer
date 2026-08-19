// Package parser implements the ports.LogParser port: it detects JSON
// payloads, infers the severity of a log line and recognizes multiline
// continuations such as stack traces.
package parser

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// levelKeys lists the JSON field names commonly used to carry the log
// severity, checked in order.
var levelKeys = []string{"level", "severity", "loglevel", "log_level"}

// messageKeys lists the JSON field names commonly used to carry the human
// readable message, checked in order.
var messageKeys = []string{"message", "msg", "log"}

// timestampKeys lists the JSON field names commonly used to carry an
// event timestamp, checked in order.
var timestampKeys = []string{"timestamp", "time", "@timestamp", "ts"}

// continuationPattern matches lines that are very likely the continuation
// of a multiline record rather than a new one: leading whitespace, a stack
// frame ("at ..."), a Python traceback header, or a "Caused by" clause.
var continuationPattern = regexp.MustCompile(`^(\s+|\tat\s|at\s|Caused by:|Traceback \(most recent call last\)|File "|\.\.\. \d+ more)`)

// severityWordPattern extracts a leading severity token from a plain text
// log line, e.g. "2024-01-02T15:04:05Z ERROR something happened".
var severityWordPattern = regexp.MustCompile(`(?i)\b(DEBUG|DBG|TRACE|INFO|NOTICE|WARN(?:ING)?|ERR(?:OR)?|ERRO|FATAL|PANIC|CRIT(?:ICAL)?)\b`)

// Parser is the default ports.LogParser implementation.
type Parser struct{}

// New builds a ready-to-use Parser.
func New() *Parser {
	return &Parser{}
}

// IsContinuation implements ports.LogParser.
func (p *Parser) IsContinuation(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	return continuationPattern.MatchString(line)
}

// Parse implements ports.LogParser.
func (p *Parser) Parse(meta domain.LogMeta, raw string) domain.LogEntry {
	entry := domain.LogEntry{
		ContainerID:   meta.ContainerID,
		ContainerName: meta.ContainerName,
		Stream:        meta.Stream,
		Timestamp:     meta.Timestamp,
		Raw:           raw,
		Message:       raw,
		Level:         domain.SeverityUnknown,
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	if fields, ok := tryParseJSON(raw); ok {
		entry.IsJSON = true
		entry.Fields = fields
		entry.Message = extractString(fields, messageKeys, raw)
		entry.Level = extractSeverity(fields)
		if ts, ok := extractTimestamp(fields); ok {
			entry.Timestamp = ts
		}
		return entry
	}

	entry.Level = severityFromText(raw)
	return entry
}

func tryParseJSON(raw string) (map[string]any, bool) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return nil, false
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return nil, false
	}
	return fields, true
}

func extractString(fields map[string]any, keys []string, fallback string) string {
	for _, k := range keys {
		if v, ok := fields[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return fallback
}

func extractSeverity(fields map[string]any) domain.Severity {
	for _, k := range levelKeys {
		if v, ok := fields[k]; ok {
			if s, ok := v.(string); ok {
				if sev := domain.ParseSeverity(s); sev != domain.SeverityUnknown {
					return sev
				}
			}
		}
	}
	return domain.SeverityUnknown
}

func extractTimestamp(fields map[string]any) (time.Time, bool) {
	for _, k := range timestampKeys {
		v, ok := fields[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999", "2006-01-02 15:04:05"} {
				if parsed, err := time.Parse(layout, t); err == nil {
					return parsed, true
				}
			}
		case float64:
			return time.Unix(int64(t), 0).UTC(), true
		}
	}
	return time.Time{}, false
}

func severityFromText(raw string) domain.Severity {
	match := severityWordPattern.FindString(raw)
	if match == "" {
		return domain.SeverityUnknown
	}
	return domain.ParseSeverity(match)
}
