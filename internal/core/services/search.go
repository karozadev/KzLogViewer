// Package services implements the application use cases (hexagonal core)
// on top of the ports, independent of any adapter.
package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

// SearchEngine is the default ports.QueryEngine implementation. It supports
// three modes inspired by Kibana's search bar: plain substring text,
// AND-of-keywords, and regular expressions.
type SearchEngine struct{}

// NewSearchEngine builds a ready-to-use SearchEngine.
func NewSearchEngine() *SearchEngine {
	return &SearchEngine{}
}

// Compile implements ports.QueryEngine.
func (e *SearchEngine) Compile(raw string, mode ports.QueryMode) (ports.Query, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return matchAllQuery{raw: raw}, nil
	}

	switch mode {
	case ports.QueryModeText:
		return textQuery{raw: raw, needle: strings.ToLower(trimmed)}, nil
	case ports.QueryModeKeyword:
		fields := strings.Fields(trimmed)
		tokens := make([]string, len(fields))
		for i, f := range fields {
			tokens[i] = strings.ToLower(f)
		}
		return keywordQuery{raw: raw, tokens: tokens}, nil
	case ports.QueryModeRegexp:
		re, err := regexp.Compile(trimmed)
		if err != nil {
			return nil, fmt.Errorf("compile regexp query %q: %w", raw, err)
		}
		return regexpQuery{raw: raw, re: re}, nil
	default:
		return nil, fmt.Errorf("unknown query mode %v", mode)
	}
}

type matchAllQuery struct{ raw string }

func (q matchAllQuery) Match(domain.LogEntry) bool { return true }
func (q matchAllQuery) Raw() string                { return q.raw }

type textQuery struct {
	raw    string
	needle string
}

func (q textQuery) Match(e domain.LogEntry) bool {
	return strings.Contains(strings.ToLower(e.Raw), q.needle) ||
		strings.Contains(strings.ToLower(e.Message), q.needle)
}
func (q textQuery) Raw() string { return q.raw }

type keywordQuery struct {
	raw    string
	tokens []string
}

func (q keywordQuery) Match(e domain.LogEntry) bool {
	haystack := strings.ToLower(e.Raw)
	for _, t := range q.tokens {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}
func (q keywordQuery) Raw() string { return q.raw }

type regexpQuery struct {
	raw string
	re  *regexp.Regexp
}

func (q regexpQuery) Match(e domain.LogEntry) bool { return q.re.MatchString(e.Raw) }
func (q regexpQuery) Raw() string                  { return q.raw }
