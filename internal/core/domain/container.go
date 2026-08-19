package domain

// ContainerState is a coarse-grained lifecycle state for a Docker container.
type ContainerState string

const (
	ContainerRunning ContainerState = "running"
	ContainerPaused  ContainerState = "paused"
	ContainerExited  ContainerState = "exited"
	ContainerOther   ContainerState = "other"
)

// Container describes a Docker container as a source of logs.
type Container struct {
	ID    string
	Name  string
	Image string
	State ContainerState
}

// ContainerEventType identifies the kind of lifecycle change reported by
// the container watcher.
type ContainerEventType int

const (
	ContainerEventStarted ContainerEventType = iota
	ContainerEventStopped
)

// ContainerEvent is emitted whenever a container appears or disappears so
// the aggregator can attach or detach the corresponding log stream.
type ContainerEvent struct {
	Type      ContainerEventType
	Container Container
}
