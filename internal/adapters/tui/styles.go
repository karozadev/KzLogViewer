package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

var (
	colorError   = lipgloss.Color("#F05B5B")
	colorWarn    = lipgloss.Color("#F2C94C")
	colorInfo    = lipgloss.Color("#56B6F0")
	colorDebug   = lipgloss.Color("#8A8F98")
	colorUnknown = lipgloss.Color("#D0D4DC")
	colorAccent  = lipgloss.Color("#6C5CE7")
	colorMuted   = lipgloss.Color("#6B7280")
	colorBg      = lipgloss.Color("#1C1F26")

	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleStatus = lipgloss.NewStyle().Foreground(colorMuted)
	styleBanner = lipgloss.NewStyle().Bold(true).Foreground(colorBg).Background(colorWarn).Padding(0, 1)

	styleSelectedLine = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#2A2E37"))
	styleFieldKey     = lipgloss.NewStyle().Foreground(colorAccent)
	styleSearchPrompt = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
)

// severityColor maps a log severity to the color used to render it,
// keeping this presentation concern entirely inside the TUI adapter.
func severityColor(s domain.Severity) lipgloss.Color {
	switch s {
	case domain.SeverityError:
		return colorError
	case domain.SeverityWarn:
		return colorWarn
	case domain.SeverityInfo:
		return colorInfo
	case domain.SeverityDebug:
		return colorDebug
	default:
		return colorUnknown
	}
}

func severityStyle(s domain.Severity) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(severityColor(s)).Bold(s == domain.SeverityError)
}
