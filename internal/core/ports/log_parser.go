package ports

import "github.com/karozadev/KzLogViewer/internal/core/domain"

// LogParser turns a raw log line (optionally accumulated over several
// physical lines, e.g. a Java stack trace) into a normalized LogEntry. It
// detects JSON payloads and the log severity.
type LogParser interface {
	Parse(meta domain.LogMeta, raw string) domain.LogEntry

	// IsContinuation reports whether line looks like the continuation of a
	// previous multiline record (e.g. a stack trace frame) rather than the
	// start of a new log entry.
	IsContinuation(line string) bool
}
