package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// Docker implements the Runtime interface using the Docker CLI.
type Docker struct{}

// NewDocker creates a Docker runtime. Returns error if Docker is not available.
func NewDocker(ctx context.Context) (*Docker, error) {
	if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
		return nil, fmt.Errorf("cannot connect to Docker daemon: %w", err)
	}
	return &Docker{}, nil
}

// Build builds a Docker image from a build context directory.
func (d *Docker) Build(ctx context.Context, opts BuildOpts) (Image, error) {
	args := []string{"build", "-t", opts.Tag, opts.ContextDir}
	if opts.NoCache {
		args = []string{"build", "--no-cache", "-t", opts.Tag, opts.ContextDir}
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Image{}, fmt.Errorf("docker build failed: %s", stderr.String())
	}

	return Image{Tag: opts.Tag}, nil
}

// Start runs a container from an image.
func (d *Docker) Start(ctx context.Context, opts StartOpts) (Container, error) {
	args := []string{"run", "-d", "--name", opts.Name}

	if opts.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d:8080", opts.Port))
	} else {
		args = append(args, "-p", "8080")
	}

	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, opts.ImageTag)

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Container{}, fmt.Errorf("docker run failed: %s", stderr.String())
	}

	containerID := strings.TrimSpace(stdout.String())

	port, err := d.resolvePort(ctx, containerID)
	if err != nil {
		return Container{}, err
	}

	return Container{ID: containerID, Port: port}, nil
}

// Stop stops a running container.
func (d *Docker) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker stop failed: %w", err)
	}
	return nil
}

// Remove removes a container.
func (d *Docker) Remove(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rm failed: %w", err)
	}
	return nil
}

// RemoveImage removes a Docker image.
func (d *Docker) RemoveImage(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "docker", "rmi", tag)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rmi failed: %w", err)
	}
	return nil
}

// Status returns the current container status.
func (d *Docker) Status(ctx context.Context, id string) (ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format",
		`{"running":{{.State.Running}},"exit_code":{{.State.ExitCode}}}`, id)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return ContainerStatus{}, fmt.Errorf("docker inspect failed: %w", err)
	}

	var status struct {
		Running  bool `json:"running"`
		ExitCode int  `json:"exit_code"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		return ContainerStatus{}, fmt.Errorf("parsing container status: %w", err)
	}

	return ContainerStatus{
		Running:  status.Running,
		ExitCode: status.ExitCode,
	}, nil
}

// Logs returns a reader for container logs.
func (d *Docker) Logs(ctx context.Context, id string, opts LogOpts) (io.ReadCloser, error) {
	args := []string{"logs"}

	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", strconv.Itoa(opts.Tail))
	}

	args = append(args, id)

	cmd := exec.CommandContext(ctx, "docker", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating log pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting docker logs: %w", err)
	}

	return stdout, nil
}

func (d *Docker) resolvePort(ctx context.Context, containerID string) (int, error) {
	cmd := exec.CommandContext(ctx, "docker", "port", containerID, "8080")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("resolving port: %w", err)
	}

	line := strings.TrimSpace(stdout.String())
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected port format: %s", line)
	}

	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, fmt.Errorf("parsing port number: %w", err)
	}
	return port, nil
}
