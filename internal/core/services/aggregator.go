package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

// Aggregator fans in the log streams of every running container into a
// single ordered channel of domain.LogEntry, reassembling multiline
// records (e.g. stack traces) and reacting to containers starting or
// stopping.
type Aggregator struct {
	source ports.LogSource
	parser ports.LogParser

	entries chan domain.LogEntry
	errs    chan error
}

// NewAggregator builds an Aggregator. bufSize controls the buffering of the
// output channel so a slow consumer (the TUI) does not stall log sources.
func NewAggregator(source ports.LogSource, parser ports.LogParser, bufSize int) *Aggregator {
	if bufSize <= 0 {
		bufSize = 1024
	}
	return &Aggregator{
		source:  source,
		parser:  parser,
		entries: make(chan domain.LogEntry, bufSize),
		errs:    make(chan error, 16),
	}
}

// Run starts aggregating logs from every currently running container and
// keeps attaching/detaching streams as containers start and stop, until ctx
// is canceled. The returned channels are closed once every goroutine has
// exited.
func (a *Aggregator) Run(ctx context.Context) (<-chan domain.LogEntry, <-chan error) {
	go a.run(ctx)
	return a.entries, a.errs
}

func (a *Aggregator) run(ctx context.Context) {
	defer close(a.entries)
	defer close(a.errs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	cancels := make(map[string]context.CancelFunc)

	attach := func(c domain.Container) {
		mu.Lock()
		if _, exists := cancels[c.ID]; exists {
			mu.Unlock()
			return
		}
		cctx, cancel := context.WithCancel(ctx) //nolint:gosec // cancel is stored in cancels and invoked by detach or the run() cleanup on ctx.Done
		cancels[c.ID] = cancel
		mu.Unlock()

		wg.Add(1)
		go func() {
			defer wg.Done()
			a.consumeContainer(cctx, c)
		}()
	}

	detach := func(id string) {
		mu.Lock()
		defer mu.Unlock()
		if cancel, ok := cancels[id]; ok {
			cancel()
			delete(cancels, id)
		}
	}

	containers, err := a.source.ListContainers(ctx)
	if err != nil {
		a.errs <- fmt.Errorf("list containers: %w", err)
	}
	for _, c := range containers {
		attach(c)
	}

	events, watchErrs := a.source.WatchContainers(ctx)
	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, cancel := range cancels {
				cancel()
			}
			mu.Unlock()
			wg.Wait()
			return
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			switch ev.Type {
			case domain.ContainerEventStarted:
				attach(ev.Container)
			case domain.ContainerEventStopped:
				detach(ev.Container.ID)
			}
		case err, ok := <-watchErrs:
			if !ok {
				watchErrs = nil
				continue
			}
			if err != nil {
				a.trySendErr(err)
			}
		}
	}
}

// consumeContainer streams and parses the log lines of a single container,
// merging continuation lines (stack traces) into the previous entry.
func (a *Aggregator) consumeContainer(ctx context.Context, c domain.Container) {
	lines, errs := a.source.StreamLogs(ctx, c.ID)

	var pending *domain.LogEntry
	var rawBuf strings.Builder

	flush := func() {
		if pending == nil {
			return
		}
		entry := *pending
		select {
		case a.entries <- entry:
		case <-ctx.Done():
		}
		pending = nil
		rawBuf.Reset()
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case line, ok := <-lines:
			if !ok {
				flush()
				return
			}
			if pending != nil && a.parser.IsContinuation(line.Text) {
				rawBuf.WriteByte('\n')
				rawBuf.WriteString(line.Text)
				merged := *pending
				merged.Raw = rawBuf.String()
				merged.Message = merged.Message + "\n" + line.Text
				merged.Multiline = true
				pending = &merged
				continue
			}
			flush()
			entry := a.parser.Parse(line.Meta, line.Text)
			pending = &entry
			rawBuf.Reset()
			rawBuf.WriteString(line.Text)
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				a.trySendErr(fmt.Errorf("container %s: %w", c.Name, err))
			}
		}
	}
}

func (a *Aggregator) trySendErr(err error) {
	select {
	case a.errs <- err:
	default:
	}
}
