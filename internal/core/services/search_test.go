package services

import (
	"testing"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

func TestSearchEngineTextMode(t *testing.T) {
	e := NewSearchEngine()
	q, err := e.Compile("timeout", ports.QueryModeText)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !q.Match(domain.LogEntry{Raw: "connection timeout after 30s"}) {
		t.Error("expected match")
	}
	if q.Match(domain.LogEntry{Raw: "everything is fine"}) {
		t.Error("expected no match")
	}
	if q.Raw() != "timeout" {
		t.Errorf("Raw() = %q", q.Raw())
	}
}

func TestSearchEngineKeywordMode(t *testing.T) {
	e := NewSearchEngine()
	q, err := e.Compile("database timeout", ports.QueryModeKeyword)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !q.Match(domain.LogEntry{Raw: "database connection timeout"}) {
		t.Error("expected match when all tokens present")
	}
	if q.Match(domain.LogEntry{Raw: "database is healthy"}) {
		t.Error("expected no match when a token is missing")
	}
}

func TestSearchEngineRegexpMode(t *testing.T) {
	e := NewSearchEngine()
	q, err := e.Compile(`status=5\d\d`, ports.QueryModeRegexp)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !q.Match(domain.LogEntry{Raw: "request finished status=502"}) {
		t.Error("expected regexp match")
	}
	if q.Match(domain.LogEntry{Raw: "request finished status=200"}) {
		t.Error("expected no regexp match")
	}
}

func TestSearchEngineInvalidRegexp(t *testing.T) {
	e := NewSearchEngine()
	_, err := e.Compile("(unclosed", ports.QueryModeRegexp)
	if err == nil {
		t.Fatal("expected an error for invalid regexp")
	}
}

func TestSearchEngineEmptyQueryMatchesAll(t *testing.T) {
	e := NewSearchEngine()
	q, err := e.Compile("   ", ports.QueryModeText)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !q.Match(domain.LogEntry{Raw: "anything"}) {
		t.Error("expected empty query to match everything")
	}
}

func TestSearchEngineUnknownMode(t *testing.T) {
	e := NewSearchEngine()
	_, err := e.Compile("x", ports.QueryMode(99))
	if err == nil {
		t.Fatal("expected an error for unknown query mode")
	}
}
