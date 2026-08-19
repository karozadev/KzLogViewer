package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

type fakeSource struct {
	containers []domain.Container
	listErr    error
	lines      map[string][]ports.RawLine
	streamErr  map[string]error
	events     chan domain.ContainerEvent
	eventErrs  chan error
}

func newFakeSource() *fakeSource {
	return &fakeSource{
		lines:     map[string][]ports.RawLine{},
		streamErr: map[string]error{},
		events:    make(chan domain.ContainerEvent),
		eventErrs: make(chan error),
	}
}

func (f *fakeSource) ListContainers(ctx context.Context) ([]domain.Container, error) {
	return f.containers, f.listErr
}

func (f *fakeSource) StreamLogs(ctx context.Context, id string) (<-chan ports.RawLine, <-chan error) {
	out := make(chan ports.RawLine)
	errCh := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errCh)
		for _, l := range f.lines[id] {
			select {
			case out <- l:
			case <-ctx.Done():
				return
			}
		}
		if err, ok := f.streamErr[id]; ok {
			errCh <- err
		}
	}()
	return out, errCh
}

func (f *fakeSource) WatchContainers(ctx context.Context) (<-chan domain.ContainerEvent, <-chan error) {
	return f.events, f.eventErrs
}

// fakeParser treats lines prefixed with two spaces as continuations, and
// otherwise passes the raw text through untouched.
type fakeParser struct{}

func (fakeParser) Parse(meta domain.LogMeta, raw string) domain.LogEntry {
	return domain.LogEntry{
		ContainerID:   meta.ContainerID,
		ContainerName: meta.ContainerName,
		Raw:           raw,
		Message:       raw,
	}
}

func (fakeParser) IsContinuation(line string) bool {
	return strings.HasPrefix(line, "  ")
}

func collect(t *testing.T, ch <-chan domain.LogEntry, n int, timeout time.Duration) []domain.LogEntry {
	t.Helper()
	var out []domain.LogEntry
	deadline := time.After(timeout)
	for len(out) < n {
		select {
		case e, ok := <-ch:
			if !ok {
				t.Fatalf("channel closed after %d entries, wanted %d", len(out), n)
			}
			out = append(out, e)
		case <-deadline:
			t.Fatalf("timed out after %d entries, wanted %d", len(out), n)
		}
	}
	return out
}

func TestAggregatorStreamsExistingContainers(t *testing.T) {
	src := newFakeSource()
	src.containers = []domain.Container{{ID: "c1", Name: "web"}}
	src.lines["c1"] = []ports.RawLine{
		{Meta: domain.LogMeta{ContainerID: "c1", ContainerName: "web"}, Text: "hello"},
		{Meta: domain.LogMeta{ContainerID: "c1", ContainerName: "web"}, Text: "world"},
	}

	agg := NewAggregator(src, fakeParser{}, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entries, _ := agg.Run(ctx)
	got := collect(t, entries, 2, 2*time.Second)

	if got[0].Message != "hello" || got[1].Message != "world" {
		t.Errorf("got %+v", got)
	}
}

func TestAggregatorMergesMultilineEntries(t *testing.T) {
	src := newFakeSource()
	src.containers = []domain.Container{{ID: "c1", Name: "web"}}
	src.lines["c1"] = []ports.RawLine{
		{Meta: domain.LogMeta{ContainerID: "c1"}, Text: "panic: boom"},
		{Meta: domain.LogMeta{ContainerID: "c1"}, Text: "  at main.go:10"},
		{Meta: domain.LogMeta{ContainerID: "c1"}, Text: "  at main.go:20"},
		{Meta: domain.LogMeta{ContainerID: "c1"}, Text: "next entry"},
	}

	agg := NewAggregator(src, fakeParser{}, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entries, _ := agg.Run(ctx)
	got := collect(t, entries, 2, 2*time.Second)

	if !got[0].Multiline {
		t.Error("expected first entry to be flagged multiline")
	}
	want := "panic: boom\n  at main.go:10\n  at main.go:20"
	if got[0].Message != want {
		t.Errorf("merged message = %q, want %q", got[0].Message, want)
	}
	if got[1].Message != "next entry" {
		t.Errorf("second entry = %q", got[1].Message)
	}
}

func TestAggregatorAttachesOnStartEvent(t *testing.T) {
	src := newFakeSource()
	src.lines["c2"] = []ports.RawLine{
		{Meta: domain.LogMeta{ContainerID: "c2"}, Text: "new container log"},
	}

	agg := NewAggregator(src, fakeParser{}, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entries, _ := agg.Run(ctx)

	src.events <- domain.ContainerEvent{
		Type:      domain.ContainerEventStarted,
		Container: domain.Container{ID: "c2", Name: "worker"},
	}

	got := collect(t, entries, 1, 2*time.Second)
	if got[0].Message != "new container log" {
		t.Errorf("got %+v", got)
	}
}

func TestAggregatorPropagatesListError(t *testing.T) {
	src := newFakeSource()
	src.listErr = errors.New("docker unreachable")

	agg := NewAggregator(src, fakeParser{}, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, errs := agg.Run(ctx)

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected an error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestAggregatorClosesChannelsOnCancel(t *testing.T) {
	src := newFakeSource()
	agg := NewAggregator(src, fakeParser{}, 0)
	ctx, cancel := context.WithCancel(context.Background())

	entries, errs := agg.Run(ctx)
	cancel()

	select {
	case _, ok := <-entries:
		if ok {
			t.Fatal("expected entries channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for entries channel to close")
	}
	select {
	case _, ok := <-errs:
		if ok {
			t.Fatal("expected errs channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for errs channel to close")
	}
}
