package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

type logEntryMsg domain.LogEntry
type sourceErrMsg struct{ err error }
type entriesClosedMsg struct{}
type tickMsg time.Time
type updateCheckMsg struct {
	release ports.Release
	err     error
}

func waitForEntry(ch <-chan domain.LogEntry) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return entriesClosedMsg{}
		}
		return logEntryMsg(e)
	}
}

func waitForSourceErr(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-ch
		if !ok {
			return nil
		}
		return sourceErrMsg{err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// checkForUpdate queries checker once, bounded by a short timeout so a
// slow or unreachable network never delays the TUI beyond a few seconds.
func checkForUpdate(checker ports.UpdateChecker) tea.Cmd {
	if checker == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		release, err := checker.LatestRelease(ctx)
		return updateCheckMsg{release: release, err: err}
	}
}
