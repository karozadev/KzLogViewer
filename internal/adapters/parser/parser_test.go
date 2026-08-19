package parser

import (
	"testing"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

func TestParseJSON(t *testing.T) {
	p := New()
	meta := domain.LogMeta{ContainerID: "abc", ContainerName: "web", Stream: domain.StreamStdout}
	raw := `{"level":"error","message":"boom","timestamp":"2024-01-02T15:04:05Z","request_id":"r-1"}`

	entry := p.Parse(meta, raw)

	if !entry.IsJSON {
		t.Fatal("expected IsJSON = true")
	}
	if entry.Level != domain.SeverityError {
		t.Errorf("Level = %v, want ERROR", entry.Level)
	}
	if entry.Message != "boom" {
		t.Errorf("Message = %q, want boom", entry.Message)
	}
	if entry.Fields["request_id"] != "r-1" {
		t.Errorf("Fields[request_id] = %v, want r-1", entry.Fields["request_id"])
	}
	wantTime := time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)
	if !entry.Timestamp.Equal(wantTime) {
		t.Errorf("Timestamp = %v, want %v", entry.Timestamp, wantTime)
	}
}

func TestParseJSONAlternateKeys(t *testing.T) {
	p := New()
	meta := domain.LogMeta{}
	raw := `{"severity":"WARN","msg":"disk almost full"}`

	entry := p.Parse(meta, raw)

	if entry.Level != domain.SeverityWarn {
		t.Errorf("Level = %v, want WARN", entry.Level)
	}
	if entry.Message != "disk almost full" {
		t.Errorf("Message = %q", entry.Message)
	}
}

func TestParsePlainText(t *testing.T) {
	p := New()
	meta := domain.LogMeta{}
	raw := "2024-01-02T15:04:05Z ERROR failed to connect to database"

	entry := p.Parse(meta, raw)

	if entry.IsJSON {
		t.Fatal("expected IsJSON = false")
	}
	if entry.Level != domain.SeverityError {
		t.Errorf("Level = %v, want ERROR", entry.Level)
	}
	if entry.Message != raw {
		t.Errorf("Message = %q, want raw line", entry.Message)
	}
}

func TestParseUnknownSeverityDefaultsToNow(t *testing.T) {
	p := New()
	before := time.Now().UTC()
	entry := p.Parse(domain.LogMeta{}, "just a plain line")
	after := time.Now().UTC()

	if entry.Level != domain.SeverityUnknown {
		t.Errorf("Level = %v, want UNKNOWN", entry.Level)
	}
	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", entry.Timestamp, before, after)
	}
}

func TestParseInvalidJSONFallsBackToPlainText(t *testing.T) {
	p := New()
	raw := `{not valid json}`
	entry := p.Parse(domain.LogMeta{}, raw)

	if entry.IsJSON {
		t.Fatal("expected IsJSON = false for invalid JSON")
	}
	if entry.Message != raw {
		t.Errorf("Message = %q, want raw", entry.Message)
	}
}

func TestIsContinuation(t *testing.T) {
	p := New()
	cases := map[string]bool{
		"    at com.example.Foo.bar(Foo.java:42)":   true,
		"Caused by: java.lang.NullPointerException": true,
		`Traceback (most recent call last):`:        true,
		`  File "app.py", line 10, in <module>`:     true,
		"... 12 more":                               true,
		"":                                          false,
		"a new log line":                            false,
		"2024 INFO normal":                          false,
	}
	for line, want := range cases {
		if got := p.IsContinuation(line); got != want {
			t.Errorf("IsContinuation(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestExtractTimestampUnixSeconds(t *testing.T) {
	p := New()
	raw := `{"level":"info","message":"ok","time":1704207845}`
	entry := p.Parse(domain.LogMeta{}, raw)
	if entry.Timestamp.Unix() != 1704207845 {
		t.Errorf("Timestamp.Unix() = %d, want 1704207845", entry.Timestamp.Unix())
	}
}
