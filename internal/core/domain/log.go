// Package domain contains the core business entities of KzLogViewer.
// It has no dependency on Docker, Bubbletea, or any other technical detail.
package domain

import (
	"strings"
	"time"
)

// Severity represents the log level of a log entry.
type Severity int

const (
	SeverityUnknown Severity = iota
	SeverityDebug
	SeverityInfo
	SeverityWarn
	SeverityError
)

// String returns the canonical textual representation of the severity.
func (s Severity) String() string {
	switch s {
	case SeverityDebug:
		return "DEBUG"
	case SeverityInfo:
		return "INFO"
	case SeverityWarn:
		return "WARN"
	case SeverityError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseSeverity infers a Severity from a free-form token such as a JSON
// "level" field or a word found in a raw log line (e.g. "ERRO", "WARNING").
func ParseSeverity(raw string) Severity {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG", "DBG", "TRACE":
		return SeverityDebug
	case "INFO", "INFORMATION", "NOTICE":
		return SeverityInfo
	case "WARN", "WARNING":
		return SeverityWarn
	case "ERROR", "ERR", "ERRO", "FATAL", "PANIC", "CRITICAL", "CRIT":
		return SeverityError
	default:
		return SeverityUnknown
	}
}

// Stream identifies the origin file descriptor of a log line.
type Stream string

const (
	StreamStdout Stream = "stdout"
	StreamStderr Stream = "stderr"
)

// LogMeta carries the metadata attached to a raw log line by the source
// adapter (Docker) before it is handed to the parser.
type LogMeta struct {
	ContainerID   string
	ContainerName string
	Stream        Stream
	Timestamp     time.Time
}

// LogEntry is the normalized representation of a single log record, whether
// it originated from a single line or was reassembled from a multiline
// stack trace.
type LogEntry struct {
	ContainerID   string
	ContainerName string
	Stream        Stream
	Timestamp     time.Time
	Level         Severity
	Message       string
	Raw           string
	Fields        map[string]any
	IsJSON        bool
	Multiline     bool
}

// Matches reports whether the entry belongs to the given container name,
// used by services that need a cheap containment check without importing
// the search engine.
func (e LogEntry) Matches(containerName string) bool {
	return containerName == "" || e.ContainerName == containerName
}
