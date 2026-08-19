package tui

import (
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case logEntryMsg:
		if !m.paused {
			m.appendEntry(domain.LogEntry(msg))
		}
		return m, waitForEntry(m.entriesCh)

	case sourceErrMsg:
		m.lastErr = msg.err
		return m, waitForSourceErr(m.errsCh)

	case entriesClosedMsg:
		return m, nil

	case tickMsg:
		return m, tick()

	case updateCheckMsg:
		m.updateChecked = true
		if msg.err == nil && msg.release.Version != "" {
			m.updateRelease = &msg.release
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) appendEntry(e domain.LogEntry) {
	m.entries = append(m.entries, e)
	m.heatmap.Add(e)
	m.containers[e.ContainerName] = struct{}{}

	if len(m.entries) > m.maxEntries {
		drop := len(m.entries) - m.maxEntries
		trimmed := make([]domain.LogEntry, len(m.entries)-drop)
		copy(trimmed, m.entries[drop:])
		m.entries = trimmed
		m.applyFilter()
		return
	}

	if m.matches(e) {
		m.filtered = append(m.filtered, len(m.entries)-1)
		if m.autoscroll {
			m.cursor = len(m.filtered) - 1
		}
	}
}

func (m *Model) matches(e domain.LogEntry) bool {
	if m.containerFilter != "" && e.ContainerName != m.containerFilter {
		return false
	}
	if m.compiledQuery == nil {
		return true
	}
	return m.compiledQuery.Match(e)
}

func (m *Model) applyFilter() {
	m.filtered = m.filtered[:0]
	for i, e := range m.entries {
		if m.matches(e) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.autoscroll || m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchFocused {
		return m.handleSearchKey(msg)
	}

	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Search):
		m.searchFocused = true
		m.searchInput.SetValue(m.query)
		m.searchInput.Focus()
		return m, nil
	case key.Matches(msg, keys.CycleMode):
		m.queryMode = (m.queryMode + 1) % 3
		m.recompileQuery()
		return m, nil
	case key.Matches(msg, keys.NextContainer):
		m.containerFilter = nextContainer(m.containers, m.containerFilter)
		m.applyFilter()
		return m, nil
	case key.Matches(msg, keys.TogglePause):
		m.paused = !m.paused
		return m, nil
	case key.Matches(msg, keys.Enter):
		if len(m.filtered) > 0 {
			if m.mode == viewList {
				m.mode = viewDetail
			} else {
				m.mode = viewList
			}
		}
		return m, nil
	case key.Matches(msg, keys.Escape):
		m.mode = viewList
		return m, nil
	case key.Matches(msg, keys.Up):
		m.moveCursor(-1)
		return m, nil
	case key.Matches(msg, keys.Down):
		m.moveCursor(1)
		return m, nil
	case key.Matches(msg, keys.Top):
		m.autoscroll = false
		m.cursor = 0
		return m, nil
	case key.Matches(msg, keys.Bottom):
		m.autoscroll = true
		m.cursor = len(m.filtered) - 1
		return m, nil
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	m.autoscroll = m.cursor == len(m.filtered)-1
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.query = m.searchInput.Value()
		m.searchFocused = false
		m.searchInput.Blur()
		m.recompileQuery()
		return m, nil
	case "esc":
		m.searchFocused = false
		m.searchInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m *Model) recompileQuery() {
	q, err := m.search.Compile(m.query, m.queryMode)
	if err != nil {
		m.lastErr = err
		return
	}
	m.compiledQuery = q
	m.applyFilter()
}

func nextContainer(containers map[string]struct{}, current string) string {
	names := make([]string, 0, len(containers)+1)
	names = append(names, "")
	for name := range containers {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, n := range names {
		if n == current {
			return names[(i+1)%len(names)]
		}
	}
	return ""
}
