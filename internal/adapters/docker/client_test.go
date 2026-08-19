package docker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

type fakeAPI struct {
	containers []container.Summary
	logs       []byte
	logsErr    error
	events     chan events.Message
	eventErrs  chan error
}

func (f *fakeAPI) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	if options.Filters.Len() == 0 {
		return f.containers, nil
	}
	ids := options.Filters.Get("id")
	if len(ids) == 0 {
		return f.containers, nil
	}
	var out []container.Summary
	for _, c := range f.containers {
		if c.ID == ids[0] {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeAPI) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	if f.logsErr != nil {
		return nil, f.logsErr
	}
	return io.NopCloser(bytes.NewReader(f.logs)), nil
}

func (f *fakeAPI) Events(ctx context.Context, options events.ListOptions) (<-chan events.Message, <-chan error) {
	return f.events, f.eventErrs
}

func TestListContainers(t *testing.T) {
	api := &fakeAPI{containers: []container.Summary{
		{ID: "abc123", Names: []string{"/web"}, Image: "nginx:latest", State: "running"},
		{ID: "def456", Names: []string{"/db"}, Image: "postgres:16", State: "exited"},
	}}
	c := newWithAPI(api)

	got, err := c.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}
	want := []domain.Container{
		{ID: "abc123", Name: "web", Image: "nginx:latest", State: domain.ContainerRunning},
		{ID: "def456", Name: "db", Image: "postgres:16", State: domain.ContainerExited},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d containers, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("container %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestWatchContainers(t *testing.T) {
	api := &fakeAPI{
		events:    make(chan events.Message, 2),
		eventErrs: make(chan error, 1),
	}
	c := newWithAPI(api)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evCh, errCh := c.WatchContainers(ctx)

	api.events <- events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionStart,
		Actor:  events.Actor{ID: "abc123", Attributes: map[string]string{"name": "web", "image": "nginx"}},
	}
	select {
	case ev := <-evCh:
		if ev.Type != domain.ContainerEventStarted || ev.Container.Name != "web" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for start event")
	}

	api.events <- events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionDie,
		Actor:  events.Actor{ID: "abc123", Attributes: map[string]string{"name": "web"}},
	}
	select {
	case ev := <-evCh:
		if ev.Type != domain.ContainerEventStopped {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stop event")
	}

	close(api.events)
}

func TestWatchContainersPropagatesError(t *testing.T) {
	api := &fakeAPI{
		events:    make(chan events.Message),
		eventErrs: make(chan error, 1),
	}
	c := newWithAPI(api)

	_, errCh := c.WatchContainers(context.Background())
	api.eventErrs <- errors.New("boom")

	select {
	case err := <-errCh:
		if err == nil || err.Error() != "boom" {
			t.Fatalf("got err %v, want boom", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestStreamLogsDemultiplexesAndParsesTimestamps(t *testing.T) {
	var buf bytes.Buffer
	stdout := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
	stderr := stdcopy.NewStdWriter(&buf, stdcopy.Stderr)

	ts := "2024-01-02T15:04:05.000000000Z"
	if _, err := stdout.Write([]byte(ts + " hello from stdout\n")); err != nil {
		t.Fatalf("write stdout fixture: %v", err)
	}
	if _, err := stderr.Write([]byte(ts + " hello from stderr\n")); err != nil {
		t.Fatalf("write stderr fixture: %v", err)
	}

	api := &fakeAPI{
		containers: []container.Summary{{ID: "abc123", Names: []string{"/web"}}},
		logs:       buf.Bytes(),
	}
	c := newWithAPI(api)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	lines, errs := c.StreamLogs(ctx, "abc123")

	var got []string
	streams := map[domain.Stream]bool{}
loop:
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				break loop
			}
			got = append(got, line.Text)
			streams[line.Meta.Stream] = true
			if line.Meta.ContainerName != "web" {
				t.Errorf("ContainerName = %q, want web", line.Meta.ContainerName)
			}
			if line.Meta.Timestamp.Year() != 2024 {
				t.Errorf("Timestamp = %v, want year 2024", line.Meta.Timestamp)
			}
		case err := <-errs:
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for log lines")
		}
	}

	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2: %v", len(got), got)
	}
	if !streams[domain.StreamStdout] || !streams[domain.StreamStderr] {
		t.Fatalf("expected both stdout and stderr, got %+v", streams)
	}
}

func TestStreamLogsPropagatesOpenError(t *testing.T) {
	api := &fakeAPI{logsErr: errors.New("no such container")}
	c := newWithAPI(api)

	lines, errs := c.StreamLogs(context.Background(), "missing")

	select {
	case _, ok := <-lines:
		if ok {
			t.Fatal("expected lines channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected an error")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestSplitTimestamp(t *testing.T) {
	ts, text := splitTimestamp("2024-01-02T15:04:05.123456789Z some message")
	if text != "some message" {
		t.Errorf("text = %q, want %q", text, "some message")
	}
	if ts.IsZero() {
		t.Errorf("expected a parsed timestamp")
	}

	_, text = splitTimestamp("not a timestamp at all")
	if text != "not a timestamp at all" {
		t.Errorf("fallback text = %q", text)
	}
}
