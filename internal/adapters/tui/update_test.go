package tui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
	"github.com/karozadev/KzLogViewer/internal/core/services"
)

func newTestModel() Model {
	entries := make(chan domain.LogEntry)
	errs := make(chan error)
	m := NewModel(entries, errs, services.NewSearchEngine(), services.NewHeatmapBuilder(time.Minute), nil, "v1.0.0")
	m.width, m.height = 80, 24
	return m
}

func sendEntry(t *testing.T, m Model, e domain.LogEntry) Model {
	t.Helper()
	updated, _ := m.Update(logEntryMsg(e))
	return updated.(Model)
}

func TestUpdateAppendsLogEntry(t *testing.T) {
	m := newTestModel()
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "hello", Timestamp: time.Now()})

	if len(m.entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(m.entries))
	}
	if len(m.filtered) != 1 {
		t.Fatalf("got %d filtered, want 1", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (autoscroll)", m.cursor)
	}
}

func TestUpdateSkipsEntriesWhilePaused(t *testing.T) {
	m := newTestModel()
	m.paused = true
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "hello"})

	if len(m.entries) != 0 {
		t.Fatalf("got %d entries, want 0 while paused", len(m.entries))
	}
}

func TestSearchFlowFiltersEntries(t *testing.T) {
	m := newTestModel()
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "connection refused", Raw: "connection refused"})
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "all good", Raw: "all good"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	if !m.searchFocused {
		t.Fatal("expected search bar to be focused after /")
	}

	for _, r := range "refused" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.searchFocused {
		t.Fatal("expected search bar to lose focus after enter")
	}
	if len(m.filtered) != 1 {
		t.Fatalf("got %d filtered entries, want 1", len(m.filtered))
	}
	if m.entries[m.filtered[0]].Message != "connection refused" {
		t.Errorf("unexpected filtered entry: %+v", m.entries[m.filtered[0]])
	}
}

func TestCycleQueryMode(t *testing.T) {
	m := newTestModel()
	if m.queryMode != ports.QueryModeText {
		t.Fatalf("expected default mode text")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.queryMode != ports.QueryModeKeyword {
		t.Errorf("queryMode = %v, want keyword", m.queryMode)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.queryMode != ports.QueryModeRegexp {
		t.Errorf("queryMode = %v, want regexp", m.queryMode)
	}
}

func TestContainerFilterCycle(t *testing.T) {
	m := newTestModel()
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "api"})
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "db"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	first := m.containerFilter
	if first == "" {
		t.Fatal("expected a container filter to be selected")
	}
	if len(m.filtered) != 1 {
		t.Fatalf("got %d filtered, want 1 for a single container", len(m.filtered))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.containerFilter == first {
		t.Error("expected the container filter to advance")
	}
}

func TestTogglePause(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	if !m.paused {
		t.Error("expected paused = true after p")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	if m.paused {
		t.Error("expected paused = false after second p")
	}
}

func TestEnterTogglesDetailView(t *testing.T) {
	m := newTestModel()
	m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "hello"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.mode != viewDetail {
		t.Fatalf("mode = %v, want detail", m.mode)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.mode != viewList {
		t.Fatalf("mode = %v, want list after esc", m.mode)
	}
}

func TestCursorMovementClampsAndDisablesAutoscroll(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 3; i++ {
		m = sendEntry(t, m, domain.LogEntry{ContainerName: "web", Message: "line"})
	}
	if !m.autoscroll || m.cursor != 2 {
		t.Fatalf("expected autoscroll at bottom, cursor=%d autoscroll=%v", m.cursor, m.autoscroll)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 1 || m.autoscroll {
		t.Fatalf("expected cursor=1 autoscroll=false, got cursor=%d autoscroll=%v", m.cursor, m.autoscroll)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Fatalf("expected cursor clamped at 0, got %d", m.cursor)
	}
}

func TestQuitSetsQuittingAndReturnsQuitCmd(t *testing.T) {
	m := newTestModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)
	if !m.quitting {
		t.Error("expected quitting = true")
	}
	if cmd == nil {
		t.Fatal("expected a non-nil quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", cmd())
	}
}

func TestUpdateHandlesWindowSize(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(Model)
	if m.width != 100 || m.height != 40 {
		t.Errorf("width/height = %d/%d, want 100/40", m.width, m.height)
	}
}

func TestUpdateCheckMsgStoresRelease(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(updateCheckMsg{release: ports.Release{Version: "v2.0.0"}})
	m = updated.(Model)
	if m.updateRelease == nil || m.updateRelease.Version != "v2.0.0" {
		t.Errorf("expected updateRelease to be stored, got %+v", m.updateRelease)
	}
}

func TestUpdateCheckMsgIgnoresError(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(updateCheckMsg{err: errors.New("boom")})
	m = updated.(Model)
	if m.updateRelease != nil {
		t.Errorf("expected no release stored on error, got %+v", m.updateRelease)
	}
}

// TestUpdateCheckMsgIgnoresEmptyVersion covers the regression where the
// startup banner announced "a new version" even while already running the
// latest one: checkForUpdate reports no update as a zero-value release
// (err == nil, Version == ""), which must not be stored as an update.
func TestUpdateCheckMsgIgnoresEmptyVersion(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(updateCheckMsg{release: ports.Release{}})
	m = updated.(Model)
	if m.updateRelease != nil {
		t.Errorf("expected no release stored when already up to date, got %+v", m.updateRelease)
	}
}
