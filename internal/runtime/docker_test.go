package runtime

import (
	"context"
	"os/exec"
	"testing"
)

func TestNewDockerFailsWithoutDocker(t *testing.T) {
	if dockerAvailable() {
		t.Skip("docker is available, this test checks the failure path")
	}
	_, err := NewDocker(context.Background())
	if err == nil {
		t.Error("expected error when Docker is not available")
	}
}

func TestNewDockerSucceeds(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	d, err := NewDocker(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Error("expected non-nil Docker runtime")
	}
}

func dockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil //nolint:gosec // test helper only
}
