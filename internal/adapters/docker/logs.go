package docker

import (
	"bufio"
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

// tailLines bounds the backlog replayed when a stream is first attached, so
// KzLogViewer starts responsive even on chatty containers.
const tailLines = "200"

// StreamLogs implements ports.LogSource. The Docker daemon multiplexes
// stdout and stderr on a single connection when the container was not
// started with a TTY; stdcopy demultiplexes it back into two streams that
// are read line by line and tagged accordingly.
func (c *Client) StreamLogs(ctx context.Context, containerID string) (<-chan ports.RawLine, <-chan error) {
	out := make(chan ports.RawLine, 256)
	outErr := make(chan error, 1)

	reader, err := c.api.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       tailLines,
	})
	if err != nil {
		close(out)
		outErr <- err
		close(outErr)
		return out, outErr
	}

	name := c.resolveName(ctx, containerID)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	go func() {
		defer func() { _ = stdoutW.Close() }()
		defer func() { _ = stderrW.Close() }()
		defer func() { _ = reader.Close() }()
		if _, err := stdcopy.StdCopy(stdoutW, stderrW, reader); err != nil && err != io.EOF {
			select {
			case outErr <- err:
			default:
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go pumpLines(&wg, stdoutR, containerID, name, domain.StreamStdout, out)
	go pumpLines(&wg, stderrR, containerID, name, domain.StreamStderr, out)

	go func() {
		wg.Wait()
		close(out)
		close(outErr)
	}()

	return out, outErr
}

// pumpLines reads demultiplexed log lines, strips the RFC3339Nano
// timestamp Docker prepends to each of them, and forwards a ports.RawLine.
func pumpLines(wg *sync.WaitGroup, r io.Reader, containerID, containerName string, stream domain.Stream, out chan<- ports.RawLine) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		ts, text := splitTimestamp(scanner.Text())
		out <- ports.RawLine{
			Meta: domain.LogMeta{
				ContainerID:   containerID,
				ContainerName: containerName,
				Stream:        stream,
				Timestamp:     ts,
			},
			Text: text,
		}
	}
}

// splitTimestamp separates the leading RFC3339Nano timestamp that Docker
// adds to every log line (LogsOptions.Timestamps) from the actual message.
func splitTimestamp(line string) (time.Time, string) {
	idx := strings.IndexByte(line, ' ')
	if idx <= 0 {
		return time.Now().UTC(), line
	}
	ts, err := time.Parse(time.RFC3339Nano, line[:idx])
	if err != nil {
		return time.Now().UTC(), line
	}
	return ts, line[idx+1:]
}
