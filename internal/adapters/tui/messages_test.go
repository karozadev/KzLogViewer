package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

func TestWaitForEntryDeliversMessage(t *testing.T) {
	ch := make(chan domain.LogEntry, 1)
	ch <- domain.LogEntry{Message: "hi"}

	msg := waitForEntry(ch)()
	entry, ok := msg.(logEntryMsg)
	if !ok {
		t.Fatalf("got %T, want logEntryMsg", msg)
	}
	if entry.Message != "hi" {
		t.Errorf("Message = %q, want hi", entry.Message)
	}
}

func TestWaitForEntryClosedChannel(t *testing.T) {
	ch := make(chan domain.LogEntry)
	close(ch)

	msg := waitForEntry(ch)()
	if _, ok := msg.(entriesClosedMsg); !ok {
		t.Fatalf("got %T, want entriesClosedMsg", msg)
	}
}

func TestWaitForSourceErrDeliversMessage(t *testing.T) {
	ch := make(chan error, 1)
	ch <- errors.New("boom")

	msg := waitForSourceErr(ch)()
	errMsg, ok := msg.(sourceErrMsg)
	if !ok {
		t.Fatalf("got %T, want sourceErrMsg", msg)
	}
	if errMsg.err.Error() != "boom" {
		t.Errorf("err = %v", errMsg.err)
	}
}

func TestWaitForSourceErrClosedChannel(t *testing.T) {
	ch := make(chan error)
	close(ch)

	if msg := waitForSourceErr(ch)(); msg != nil {
		t.Errorf("expected nil message on closed channel, got %v", msg)
	}
}

func TestTickReturnsTickMsg(t *testing.T) {
	cmd := tick()
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}
	if _, ok := cmd().(tickMsg); !ok {
		t.Errorf("got %T, want tickMsg", cmd())
	}
}

type fakeChecker struct {
	release  ports.Release
	releases []ports.Release
	err      error
}

func (f fakeChecker) LatestRelease(ctx context.Context) (ports.Release, error) {
	return f.release, f.err
}

func (f fakeChecker) ReleasesSince(ctx context.Context, currentVersion string) ([]ports.Release, error) {
	return f.releases, f.err
}

func TestCheckForUpdateReturnsNewestSkippedRelease(t *testing.T) {
	checker := fakeChecker{releases: []ports.Release{
		{Version: "v9.9.0"},
		{Version: "v9.9.9"},
	}}
	cmd := checkForUpdate(checker, "v9.8.0")
	msg, ok := cmd().(updateCheckMsg)
	if !ok {
		t.Fatalf("got %T, want updateCheckMsg", msg)
	}
	if msg.release.Version != "v9.9.9" {
		t.Errorf("Version = %q, want the newest of the skipped releases", msg.release.Version)
	}
}

func TestCheckForUpdateNoNewerRelease(t *testing.T) {
	checker := fakeChecker{releases: nil}
	cmd := checkForUpdate(checker, "v0.1.0")
	msg, ok := cmd().(updateCheckMsg)
	if !ok {
		t.Fatalf("got %T, want updateCheckMsg", msg)
	}
	if msg.release.Version != "" || msg.err != nil {
		t.Errorf("expected an empty result when already up to date, got %+v", msg)
	}
}

func TestCheckForUpdatePropagatesError(t *testing.T) {
	checker := fakeChecker{err: errors.New("network unreachable")}
	cmd := checkForUpdate(checker, "v0.1.0")
	msg, ok := cmd().(updateCheckMsg)
	if !ok {
		t.Fatalf("got %T, want updateCheckMsg", msg)
	}
	if msg.err == nil {
		t.Error("expected the error to be propagated")
	}
}

func TestCheckForUpdateNilChecker(t *testing.T) {
	if cmd := checkForUpdate(nil, "v0.1.0"); cmd != nil {
		t.Error("expected nil command for nil checker")
	}
}
