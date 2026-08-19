package domain

import "testing"

func TestParseSeverity(t *testing.T) {
	cases := map[string]Severity{
		"debug":    SeverityDebug,
		"DBG":      SeverityDebug,
		"trace":    SeverityDebug,
		"info":     SeverityInfo,
		"notice":   SeverityInfo,
		"warn":     SeverityWarn,
		"WARNING":  SeverityWarn,
		"error":    SeverityError,
		"ERRO":     SeverityError,
		"fatal":    SeverityError,
		"panic":    SeverityError,
		"critical": SeverityError,
		"":         SeverityUnknown,
		"nope":     SeverityUnknown,
	}
	for in, want := range cases {
		if got := ParseSeverity(in); got != want {
			t.Errorf("ParseSeverity(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSeverityString(t *testing.T) {
	cases := map[Severity]string{
		SeverityDebug:   "DEBUG",
		SeverityInfo:    "INFO",
		SeverityWarn:    "WARN",
		SeverityError:   "ERROR",
		SeverityUnknown: "UNKNOWN",
		Severity(99):    "UNKNOWN",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("Severity(%d).String() = %q, want %q", in, got, want)
		}
	}
}

func TestLogEntryMatches(t *testing.T) {
	e := LogEntry{ContainerName: "web"}
	if !e.Matches("") {
		t.Error("empty filter should match any container")
	}
	if !e.Matches("web") {
		t.Error("expected match for same container name")
	}
	if e.Matches("db") {
		t.Error("expected no match for a different container name")
	}
}
