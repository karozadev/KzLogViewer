// Package tui implements the KzLogViewer terminal user interface on top of
// Bubbletea and Lipgloss. It is the primary adapter driving the core
// application services; it never talks to Docker or GitHub directly.
package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
	"github.com/karozadev/KzLogViewer/internal/core/services"
)

type viewMode int

const (
	viewList viewMode = iota
	viewDetail
)

const defaultMaxEntries = 5000

// Model is the Bubbletea model driving the whole TUI. It holds the
// in-memory log buffer, the current filter, and the layout state; it reads
// new entries from the channels produced by the aggregator service.
type Model struct {
	entriesCh <-chan domain.LogEntry
	errsCh    <-chan error

	search  ports.QueryEngine
	heatmap *services.HeatmapBuilder
	checker ports.UpdateChecker

	currentVersion string

	entries    []domain.LogEntry
	filtered   []int
	maxEntries int

	containers      map[string]struct{}
	containerFilter string

	query         string
	queryMode     ports.QueryMode
	compiledQuery ports.Query
	searchInput   textinput.Model
	searchFocused bool

	mode       viewMode
	cursor     int
	offset     int
	autoscroll bool
	paused     bool

	width, height int

	lastErr       error
	updateRelease *ports.Release
	updateChecked bool

	quitting bool
}

// NewModel builds the initial TUI model. entries and errs are the channels
// produced by services.Aggregator.Run; search compiles user queries;
// heatmap accumulates density buckets fed by every appended entry; checker
// may be nil to disable the startup update check.
func NewModel(entries <-chan domain.LogEntry, errs <-chan error, search ports.QueryEngine, heatmap *services.HeatmapBuilder, checker ports.UpdateChecker, currentVersion string) Model {
	input := textinput.New()
	input.Placeholder = "search logs (text, keyword, or /regexp/)"
	input.Prompt = "search> "
	input.CharLimit = 256

	return Model{
		entriesCh:      entries,
		errsCh:         errs,
		search:         search,
		heatmap:        heatmap,
		checker:        checker,
		currentVersion: currentVersion,
		maxEntries:     defaultMaxEntries,
		containers:     map[string]struct{}{},
		searchInput:    input,
		autoscroll:     true,
		mode:           viewList,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForEntry(m.entriesCh),
		waitForSourceErr(m.errsCh),
		tick(),
		checkForUpdate(m.checker, m.currentVersion),
	)
}
