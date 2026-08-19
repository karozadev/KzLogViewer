// Package ports defines the interfaces (hexagonal ports) through which the
// core domain and application services communicate with the outside world.
// Adapters implement these interfaces; the core never imports an adapter.
package ports

import (
	"context"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// RawLine is a single unparsed log line emitted by a LogSource, still
// carrying enough metadata for the LogParser to normalize it.
type RawLine struct {
	Meta domain.LogMeta
	Text string
}

// LogSource is the port through which the application discovers containers
// and streams their raw log lines. It is implemented by the Docker adapter.
type LogSource interface {
	// ListContainers returns the containers currently visible to the source.
	ListContainers(ctx context.Context) ([]domain.Container, error)

	// StreamLogs streams raw log lines for a single container until ctx is
	// canceled or the container stops. The error channel receives at most
	// one error and is then closed, followed by the closing of the line
	// channel.
	StreamLogs(ctx context.Context, containerID string) (<-chan RawLine, <-chan error)

	// WatchContainers streams container lifecycle events (start/stop) so the
	// aggregator can attach or detach log streams dynamically.
	WatchContainers(ctx context.Context) (<-chan domain.ContainerEvent, <-chan error)
}
