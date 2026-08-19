package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

// TestViewFillsExactlyTerminalHeight guards against a regression where
// View() rendered one line short of m.height. Bubbletea's alt-screen
// renderer diffs frames by line count; any mismatch between the reported
// height and the actual number of lines produced left a stale row on
// screen that only became visible once content stopped changing every
// frame (e.g. right after leaving the search box), showing up as a
// "duplicated" status bar line.
func TestViewFillsExactlyTerminalHeight(t *testing.T) {
	for _, height := range []int{10, 24, 35, 40} {
		for _, withBanner := range []bool{false, true} {
			for _, detail := range []bool{false, true} {
				m := newTestModel()
				m.width, m.height = 100, height
				if withBanner {
					release := ports.Release{Version: "v9.9.9"}
					m.updateRelease = &release
				}
				m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "hello"})
				if detail {
					updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
					m = updated.(Model)
				}

				got := strings.Count(m.View(), "\n") + 1
				if got != height {
					t.Errorf("height=%d banner=%v detail=%v: View() produced %d lines, want %d",
						height, withBanner, detail, got, height)
				}
			}
		}
	}
}

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

func TestStatusBarShowsNormalModeByDefault(t *testing.T) {
	m := newTestModel()
	out := stripANSI(m.View())
	if !strings.Contains(out, "NORMAL") {
		t.Errorf("expected NORMAL mode badge, got:\n%s", out)
	}
	if strings.Contains(out, "INSERT") {
		t.Errorf("did not expect INSERT mode badge while not searching, got:\n%s", out)
	}
}

func TestStatusBarShowsInsertModeWhileSearching(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)

	out := stripANSI(m.View())
	if !strings.Contains(out, "INSERT") {
		t.Errorf("expected INSERT mode badge while searching, got:\n%s", out)
	}
	if strings.Contains(out, "NORMAL") {
		t.Errorf("did not expect NORMAL mode badge while searching, got:\n%s", out)
	}
}

func TestStatusBarReturnsToNormalAfterSearch(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	out := stripANSI(m.View())
	if !strings.Contains(out, "NORMAL") {
		t.Errorf("expected NORMAL mode badge after esc, got:\n%s", out)
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
