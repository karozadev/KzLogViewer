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
// It uses ReleasesSince rather than LatestRelease so the comparison against
// currentVersion happens once, semver-aware, in the adapter: LatestRelease
// alone would report "a new version" even when it is the version already
// running (e.g. tag "v0.1.0" vs. the ldflags-injected "0.1.0").
func checkForUpdate(checker ports.UpdateChecker, currentVersion string) tea.Cmd {
	if checker == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		releases, err := checker.ReleasesSince(ctx, currentVersion)
		if err != nil {
			return updateCheckMsg{err: err}
		}
		if len(releases) == 0 {
			return updateCheckMsg{}
		}
		return updateCheckMsg{release: releases[len(releases)-1]}
	}
}
