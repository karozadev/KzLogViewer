package tui

import (
	"fmt"
	"strings"

	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "starting KzLogViewer..."
	}

	var sections []string

	if m.updateRelease != nil {
		sections = append(sections, styleBanner.Render(fmt.Sprintf(
			"new version available: %s (current %s) - run `kzlogviewer update`",
			m.updateRelease.Version, m.currentVersion)))
	}

	sections = append(sections, m.renderHeader())
	sections = append(sections, renderHeatmap(m.heatmap.Last(m.width), m.width))
	sections = append(sections, m.renderSearchBar())

	bodyHeight := m.height - lineCount(sections) - 2 // reserve the status bar
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	if m.mode == viewDetail && len(m.filtered) > 0 {
		sections = append(sections, m.renderDetailBody(bodyHeight))
	} else {
		sections = append(sections, m.renderListBody(bodyHeight))
	}

	sections = append(sections, m.renderStatusBar())

	return strings.Join(sections, "\n")
}

func lineCount(sections []string) int {
	n := 0
	for _, s := range sections {
		n += strings.Count(s, "\n") + 1
	}
	return n
}

func (m Model) renderHeader() string {
	title := styleHeader.Render("KzLogViewer")
	sub := styleStatus.Render(fmt.Sprintf(" - %d containers - %d/%d entries", len(m.containers), len(m.filtered), len(m.entries)))
	return title + sub
}

func (m Model) renderSearchBar() string {
	mode := queryModeLabel(m.queryMode)
	filter := "all containers"
	if m.containerFilter != "" {
		filter = m.containerFilter
	}
	promptStyle := styleSearchPrompt
	if m.searchFocused {
		promptStyle = promptStyle.Foreground(colorInsert)
	}
	prompt := promptStyle.Render(fmt.Sprintf("[%s | %s] ", mode, filter))
	return prompt + m.searchInput.View()
}

func queryModeLabel(mode ports.QueryMode) string {
	switch mode {
	case ports.QueryModeKeyword:
		return "keyword"
	case ports.QueryModeRegexp:
		return "regexp"
	default:
		return "text"
	}
}

func (m Model) renderListBody(height int) string {
	m.offset = scrollOffset(m.offset, m.cursor, len(m.filtered), height)
	return renderList(m.entries, m.filtered, m.cursor, m.offset, height, m.width)
}

func (m Model) renderDetailBody(height int) string {
	idx := m.filtered[m.cursor]
	body := renderDetail(m.entries[idx], m.width)
	lines := strings.Split(body, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderStatusBar renders the bottom bar, always led by a mode badge
// (NORMAL or INSERT) so it is unambiguous whether keystrokes are
// interpreted as commands or typed into the search box.
func (m Model) renderStatusBar() string {
	badge := styleModeNormal.Render("NORMAL")
	help := "/ search   tab mode   c container   enter detail   p pause   q quit"
	if m.searchFocused {
		badge = styleModeInsert.Render("INSERT")
		help = "enter confirm search   esc cancel"
	}

	state := "streaming"
	if m.paused {
		state = "paused"
	}
	rest := fmt.Sprintf(" %s | %s", state, help)
	if m.lastErr != nil {
		rest = fmt.Sprintf(" error: %v | %s", m.lastErr, help)
	}
	return badge + styleStatus.Render(rest)
}
