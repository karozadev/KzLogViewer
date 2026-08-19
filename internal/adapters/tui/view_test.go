package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

func TestViewBeforeWindowSizeShowsStartupMessage(t *testing.T) {
	m := newTestModel()
	m.width = 0
	if !strings.Contains(m.View(), "starting") {
		t.Errorf("expected startup placeholder, got %q", m.View())
	}
}

func TestViewQuittingIsEmpty(t *testing.T) {
	m := newTestModel()
	m.quitting = true
	if got := m.View(); got != "" {
		t.Errorf("expected empty view while quitting, got %q", got)
	}
}

func TestViewListModeRendersEntries(t *testing.T) {
	m := newTestModel()
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "hello world"})

	out := stripANSI(m.View())
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected message in view, got:\n%s", out)
	}
	if !strings.Contains(out, "KzLogViewer") {
		t.Errorf("expected header in view, got:\n%s", out)
	}
}

func TestViewDetailModeRendersFields(t *testing.T) {
	m := newTestModel()
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "hello", Fields: map[string]any{"k": "v"}, IsJSON: true})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	out := stripANSI(m.View())
	if !strings.Contains(out, "fields") {
		t.Errorf("expected detail fields section, got:\n%s", out)
	}
}

func TestViewShowsUpdateBanner(t *testing.T) {
	m := newTestModel()
	release := ports.Release{Version: "v9.9.9"}
	m.updateRelease = &release

	out := stripANSI(m.View())
	if !strings.Contains(out, "v9.9.9") {
		t.Errorf("expected update banner with new version, got:\n%s", out)
	}
}

func TestQueryModeLabel(t *testing.T) {
	cases := map[ports.QueryMode]string{
		ports.QueryModeText:    "text",
		ports.QueryModeKeyword: "keyword",
		ports.QueryModeRegexp:  "regexp",
	}
	for mode, want := range cases {
		if got := queryModeLabel(mode); got != want {
			t.Errorf("queryModeLabel(%v) = %q, want %q", mode, got, want)
		}
	}
}
