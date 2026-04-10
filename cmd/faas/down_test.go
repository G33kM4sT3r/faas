package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
)

func seedFunction(t *testing.T, name, containerID, imageID string, port int) {
	t.Helper()
	if err := store.Set(&state.Function{
		Name:        name,
		Path:        "/tmp/" + name,
		Language:    "python",
		ContainerID: containerID,
		ImageID:     imageID,
		Port:        port,
		Status:      state.StatusHealthy,
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRunDownSingleFunction(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "hello", "abc123", "faas-hello:tag", 5000)

	fake := &fakeRuntime{}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	downAll = false
	downKeepImage = false
	t.Cleanup(func() { downAll = false; downKeepImage = false })

	if err := runDown(cmd, []string{"hello"}); err != nil {
		t.Fatalf("runDown failed: %v", err)
	}
	if len(fake.StopCalls) != 1 || fake.StopCalls[0] != "abc123" {
		t.Errorf("expected Stop([abc123]), got %v", fake.StopCalls)
	}
	if len(fake.RemoveCalls) != 1 {
		t.Errorf("expected one Remove call, got %v", fake.RemoveCalls)
	}
	if _, err := store.Get("hello"); err == nil {
		t.Error("expected state record to be removed after successful down")
	}
}

func TestRunDownAllFunctions(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "a", "id-a", "img-a", 5001)
	seedFunction(t, "b", "id-b", "img-b", 5002)

	fake := &fakeRuntime{}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	downAll = true
	downKeepImage = true // skip RemoveImage path
	t.Cleanup(func() { downAll = false; downKeepImage = false })

	if err := runDown(cmd, nil); err != nil {
		t.Fatalf("runDown --all failed: %v", err)
	}
	if len(fake.StopCalls) != 2 {
		t.Errorf("expected 2 Stop calls, got %d", len(fake.StopCalls))
	}
	if len(fake.RemoveCalls) != 2 {
		t.Errorf("expected 2 Remove calls, got %d", len(fake.RemoveCalls))
	}
	if fns, _ := store.List(); len(fns) != 0 {
		t.Errorf("expected empty store after --all, got %d functions", len(fns))
	}
}

func TestRunDownNotFound(t *testing.T) {
	setupCmdEnv(t)

	fake := &fakeRuntime{}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runDown(cmd, []string{"ghost"})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !strings.Contains(err.Error(), `"ghost" not found`) {
		t.Errorf("expected 'ghost not found' in error, got: %v", err)
	}
	if len(fake.StopCalls) != 0 {
		t.Errorf("expected no Stop calls for not-found, got %v", fake.StopCalls)
	}
}

// TestRunDownKeepsStateOnRemoveFailure is the regression test for the
// audit finding that stopAndRemove was wiping state even when docker
// failed to remove the container. State must persist so the user can retry.
func TestRunDownKeepsStateOnRemoveFailure(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "stuck", "id-stuck", "img-stuck", 5003)

	errBoom := errors.New("docker rm: container is in use")
	fake := &fakeRuntime{
		RemoveFn: func(ctx context.Context, id string) error { return errBoom },
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	downAll = false
	downKeepImage = true
	t.Cleanup(func() { downAll = false; downKeepImage = false })

	err := runDown(cmd, []string{"stuck"})
	if err == nil {
		t.Fatal("expected error when Remove fails, got nil")
	}
	// State must still exist so the user can retry.
	if _, err := store.Get("stuck"); err != nil {
		t.Errorf("state should be retained when Remove fails; got: %v", err)
	}
}

// TestRunDownAllAggregatesErrors verifies that --all keeps going past a
// failed function and reports all errors.
func TestRunDownAllAggregatesErrors(t *testing.T) {
	setupCmdEnv(t)
	seedFunction(t, "ok", "id-ok", "img-ok", 5004)
	seedFunction(t, "broken", "id-broken", "img-broken", 5005)

	fake := &fakeRuntime{
		RemoveFn: func(ctx context.Context, id string) error {
			if id == "id-broken" {
				return errors.New("nope")
			}
			return nil
		},
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	downAll = true
	downKeepImage = true
	t.Cleanup(func() { downAll = false; downKeepImage = false })

	err := runDown(cmd, nil)
	if err == nil {
		t.Fatal("expected aggregated error from --all, got nil")
	}
	if !strings.Contains(err.Error(), "remove broken") {
		t.Errorf("expected error to mention 'remove broken', got: %v", err)
	}
	// "ok" should be removed; "broken" should remain.
	if _, err := store.Get("ok"); err == nil {
		t.Error("expected 'ok' to be removed after successful tear-down")
	}
	if _, err := store.Get("broken"); err != nil {
		t.Errorf("expected 'broken' to remain after failed tear-down; got: %v", err)
	}
}

func TestRunDownNoArgsNoAll(t *testing.T) {
	setupCmdEnv(t)
	defer withFakeRuntime(t, &fakeRuntime{})()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	downAll = false
	t.Cleanup(func() { downAll = false })

	err := runDown(cmd, nil)
	if err == nil {
		t.Fatal("expected error when called without args or --all")
	}
	if !strings.Contains(err.Error(), "provide a function name") {
		t.Errorf("expected 'provide a function name' in error, got: %v", err)
	}
}

// Static check that runtime.Container is the import we need (silences unused).
var _ = runtime.Container{}
