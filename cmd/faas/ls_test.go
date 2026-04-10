package main

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/runtime"
)

func TestRunLsEmpty(t *testing.T) {
	setupCmdEnv(t)
	defer withFakeRuntime(t, &fakeRuntime{})()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := runLs(cmd, nil); err != nil {
		t.Errorf("runLs on empty store should not error: %v", err)
	}
}

func TestRunLsWithFunctions(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "alpha", "id-1", "img-1", 5100)
	seedFunction(t, "beta", "id-2", "img-2", 5101)

	fake := &fakeRuntime{
		StatusFn: func(ctx context.Context, id string) (runtime.ContainerStatus, error) {
			return runtime.ContainerStatus{Running: true}, nil
		},
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	lsJSON = false
	lsQuiet = false
	t.Cleanup(func() { lsJSON = false; lsQuiet = false })

	if err := runLs(cmd, nil); err != nil {
		t.Errorf("runLs failed: %v", err)
	}
}

func TestRunLsJSON(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "json-fn", "id-j", "img-j", 5102)
	defer withFakeRuntime(t, &fakeRuntime{})()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	lsJSON = true
	t.Cleanup(func() { lsJSON = false })

	if err := runLs(cmd, nil); err != nil {
		t.Errorf("runLs --json failed: %v", err)
	}
}

func TestRunLsQuiet(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "quiet-fn", "id-q", "img-q", 5103)
	defer withFakeRuntime(t, &fakeRuntime{})()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	lsQuiet = true
	t.Cleanup(func() { lsQuiet = false })

	if err := runLs(cmd, nil); err != nil {
		t.Errorf("runLs --quiet failed: %v", err)
	}
}

// TestRunLsMarksStoppedWhenContainerGone verifies the live-status sync:
// when docker says a container is not running, ls updates its state row.
func TestRunLsMarksStoppedWhenContainerGone(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "ghost", "id-ghost", "img-ghost", 5104)

	fake := &fakeRuntime{
		StatusFn: func(ctx context.Context, id string) (runtime.ContainerStatus, error) {
			return runtime.ContainerStatus{Running: false}, errors.New("no such container")
		},
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := runLs(cmd, nil); err != nil {
		t.Errorf("runLs failed: %v", err)
	}
	got, err := store.Get("ghost")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "stopped" {
		t.Errorf("expected status 'stopped' after live sync, got %q", got.Status)
	}
}
