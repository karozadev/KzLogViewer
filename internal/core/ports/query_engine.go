package ports

import "github.com/karozadev/KzLogViewer/internal/core/domain"

// QueryMode selects how a raw query string is interpreted.
type QueryMode int

const (
	// QueryModeText performs a case-insensitive substring search.
	QueryModeText QueryMode = iota
	// QueryModeKeyword requires every whitespace-separated token to be
	// present (case-insensitive, AND semantics), mirroring a simple Kibana
	// free-text search.
	QueryModeKeyword
	// QueryModeRegexp compiles the query as a regular expression.
	QueryModeRegexp
)

// Query is a compiled, ready-to-evaluate filter.
type Query interface {
	// Match reports whether the entry satisfies the query.
	Match(entry domain.LogEntry) bool
	// Raw returns the original query string.
	Raw() string
}

// QueryEngine compiles user-provided search strings into a Query.
type QueryEngine interface {
	Compile(raw string, mode QueryMode) (Query, error)
}
