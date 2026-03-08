// Package runtime defines the container runtime interface and implementations.
package runtime

import (
	"context"
	"io"
)

// Container represents a running container.
type Container struct {
	ID   string
	Port int
}

// Image represents a built container image.
type Image struct {
	ID  string
	Tag string
}

// ContainerStatus represents the current state of a container.
type ContainerStatus struct {
	Running  bool
	ExitCode int
}

// StartOpts configures container startup.
type StartOpts struct {
	ImageTag string
	Name     string
	Port     int // 0 = auto-assign
	Env      map[string]string
}

// BuildOpts configures image building.
type BuildOpts struct {
	ContextDir string
	Tag        string
	NoCache    bool
}

// LogOpts configures log streaming.
type LogOpts struct {
	Follow bool
	Tail   int // number of lines, 0 = all
}

// Runtime is the container runtime interface.
type Runtime interface {
	Build(ctx context.Context, opts BuildOpts) (Image, error)
	Start(ctx context.Context, opts StartOpts) (Container, error)
	Stop(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	RemoveImage(ctx context.Context, tag string) error
	Status(ctx context.Context, id string) (ContainerStatus, error)
	Logs(ctx context.Context, id string, opts LogOpts) (io.ReadCloser, error)
}
