// Package docker implements the ports.LogSource port on top of the
// official Docker Go SDK, talking to the local daemon through
// /var/run/docker.sock without any third-party agent.
package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/karozadev/KzLogViewer/internal/core/domain"
)

// dockerAPI is the narrow subset of *client.Client used by this adapter. It
// exists so tests can substitute a fake implementation without a running
// Docker daemon.
type dockerAPI interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error)
	Events(ctx context.Context, options events.ListOptions) (<-chan events.Message, <-chan error)
}

// Client is the Docker adapter, implementing ports.LogSource.
type Client struct {
	api dockerAPI
}

// NewFromEnvironment builds a Client from the standard Docker environment
// variables (DOCKER_HOST, DOCKER_CERT_PATH, ...), defaulting to the local
// unix socket, and negotiates the API version with the daemon.
func NewFromEnvironment() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{api: cli}, nil
}

// newWithAPI builds a Client around an arbitrary dockerAPI, used by tests.
func newWithAPI(api dockerAPI) *Client {
	return &Client{api: api}
}

// ListContainers implements ports.LogSource.
func (c *Client) ListContainers(ctx context.Context) ([]domain.Container, error) {
	summaries, err := c.api.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Container, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, toDomainContainer(s))
	}
	return out, nil
}

// WatchContainers implements ports.LogSource.
func (c *Client) WatchContainers(ctx context.Context) (<-chan domain.ContainerEvent, <-chan error) {
	out := make(chan domain.ContainerEvent)
	outErr := make(chan error, 1)

	filterArgs := filters.NewArgs(
		filters.Arg("type", string(events.ContainerEventType)),
		filters.Arg("event", string(events.ActionStart)),
		filters.Arg("event", string(events.ActionDie)),
	)
	msgs, errs := c.api.Events(ctx, events.ListOptions{Filters: filterArgs})

	go func() {
		defer close(out)
		defer close(outErr)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				ev, ok := toDomainEvent(msg)
				if !ok {
					continue
				}
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			case err, ok := <-errs:
				if !ok {
					return
				}
				if err != nil && err != io.EOF {
					select {
					case outErr <- err:
					case <-ctx.Done():
					}
					return
				}
			}
		}
	}()

	return out, outErr
}

func toDomainContainer(s container.Summary) domain.Container {
	return domain.Container{
		ID:    s.ID,
		Name:  containerName(s.Names),
		Image: s.Image,
		State: toDomainState(s.State),
	}
}

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	name := names[0]
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	return name
}

func toDomainState(state string) domain.ContainerState {
	switch state {
	case "running":
		return domain.ContainerRunning
	case "paused":
		return domain.ContainerPaused
	case "exited", "dead":
		return domain.ContainerExited
	default:
		return domain.ContainerOther
	}
}

func toDomainEvent(msg events.Message) (domain.ContainerEvent, bool) {
	name := msg.Actor.Attributes["name"]
	c := domain.Container{ID: msg.Actor.ID, Name: name, Image: msg.Actor.Attributes["image"]}

	switch msg.Action {
	case events.ActionStart:
		c.State = domain.ContainerRunning
		return domain.ContainerEvent{Type: domain.ContainerEventStarted, Container: c}, true
	case events.ActionDie:
		c.State = domain.ContainerExited
		return domain.ContainerEvent{Type: domain.ContainerEventStopped, Container: c}, true
	default:
		return domain.ContainerEvent{}, false
	}
}

// resolveName looks up the display name of a single container by ID. It is
// used by StreamLogs, which only receives an ID, to populate LogMeta.
func (c *Client) resolveName(ctx context.Context, containerID string) string {
	summaries, err := c.api.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("id", containerID)),
	})
	if err != nil || len(summaries) == 0 {
		return containerID
	}
	if name := containerName(summaries[0].Names); name != "" {
		return name
	}
	return containerID
}
