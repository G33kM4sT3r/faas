package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/config"
	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
)

func TestTearDownContainerCallsStopThenRemove(t *testing.T) {
	fake := &fakeRuntime{}
	tearDownContainer(context.Background(), fake, "abc")

	if len(fake.StopCalls) != 1 || fake.StopCalls[0] != "abc" {
		t.Errorf("stop calls: got %v, want [abc]", fake.StopCalls)
	}
	if len(fake.RemoveCalls) != 1 || fake.RemoveCalls[0] != "abc" {
		t.Errorf("remove calls: got %v, want [abc]", fake.RemoveCalls)
	}
}

func TestTearDownContainerContinuesOnStopError(t *testing.T) {
	errBoom := errors.New("boom")
	fake := &fakeRuntime{
		StopFn: func(ctx context.Context, id string) error { return errBoom },
	}
	tearDownContainer(context.Background(), fake, "abc")
	if len(fake.RemoveCalls) != 1 {
		t.Errorf("remove should still be called after stop error; got %v", fake.RemoveCalls)
	}
}

// setupCmdEnv prepares the package-level state (logger, store, $HOME, NO_COLOR)
// that doUp/runUp expect from cobra's PersistentPreRun. Returns the function
// dir for the caller to write fixtures into.
func setupCmdEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)     // faasHome() → tmp/.faas
	t.Setenv("NO_COLOR", "1") // bypass bubbletea spinner in RunWithSpinner
	store = state.New(filepath.Join(tmp, ".faas", "state.json"))
	// logger is a zerolog.Logger zero value here; calls like logger.Warn().Msg(...)
	// are safe (zero value is a no-op writer).
	return tmp
}

// writeHandlerFile drops a minimal Go handler in dir and returns its path.
func writeHandlerFile(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := []byte("package main\nfunc Handler(req map[string]any) map[string]any { return nil }\n")
	p := filepath.Join(dir, "handler.go")
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestRunUpDelegatesToDoUp verifies the runUp wrapper passes the upForce
// global through to doUp without altering the args.
func TestRunUpDelegatesToDoUp(t *testing.T) {
	tmp := setupCmdEnv(t)
	handlerPath := writeHandlerFile(t, filepath.Join(tmp, "func"))
	seedFunction(t, "handler", "stale", "img", 5400)

	defer withFakeRuntime(t, &fakeRuntime{})()

	upForce = false
	t.Cleanup(func() { upForce = false })

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runUp(cmd, []string{handlerPath})
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected runUp to refuse with 'already running' (upForce=false); got: %v", err)
	}
}

// TestDoUpRefusesAlreadyRunningWithoutForce verifies that the existing-state
// branch returns the helpful "already running" error and does NOT attempt
// any docker calls.
func TestDoUpRefusesAlreadyRunningWithoutForce(t *testing.T) {
	tmp := setupCmdEnv(t)
	handlerPath := writeHandlerFile(t, filepath.Join(tmp, "func"))
	seedFunction(t, "handler", "stale-id", "stale-img", 5300)

	fake := &fakeRuntime{}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := doUp(cmd, []string{handlerPath}, false)
	if err == nil {
		t.Fatal("expected 'already running' error, got nil")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected 'already running' in error, got: %v", err)
	}
	if len(fake.StopCalls)+len(fake.RemoveCalls) != 0 {
		t.Errorf("no docker calls expected on refusal; got Stop=%v Remove=%v", fake.StopCalls, fake.RemoveCalls)
	}
}

// TestApplyEnvOverridesRejectsMalformed is the regression test for the audit
// fix: --env values without `=` used to be silently dropped, masking typos.
func TestApplyEnvOverridesRejectsMalformed(t *testing.T) {
	cases := []struct {
		name string
		envs []string
		want string
	}{
		{"missing equals", []string{"NOEQ"}, "want KEY=VALUE"},
		{"empty key", []string{"=val"}, "key is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{}
			err := applyEnvOverrides(&cfg, tc.envs)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestApplyEnvOverridesAccepts(t *testing.T) {
	cfg := config.Config{}
	if err := applyEnvOverrides(&cfg, []string{"FOO=bar", "EMPTY="}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %q", cfg.Env["FOO"])
	}
	if v, ok := cfg.Env["EMPTY"]; !ok || v != "" {
		t.Errorf("expected EMPTY to be present and empty, got %q (ok=%v)", v, ok)
	}
}

// TestDoUpForceSurfacesTearDownError is the regression test for the audit
// fix: when force=true and tearing down the existing function fails, the
// error must surface — otherwise the next `docker run --name faas-X` fails
// with a confusing "name already in use" error.
func TestDoUpForceSurfacesTearDownError(t *testing.T) {
	tmp := setupCmdEnv(t)
	handlerPath := writeHandlerFile(t, filepath.Join(tmp, "func"))
	seedFunction(t, "handler", "stuck-id", "stuck-img", 5302)

	fake := &fakeRuntime{
		// Force the Remove step to fail so tearDown returns a non-nil error.
		RemoveFn: func(ctx context.Context, id string) error {
			return errors.New("docker rm failed: stuck")
		},
	}
	defer withFakeRuntime(t, fake)()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := doUp(cmd, []string{handlerPath}, true)
	if err == nil {
		t.Fatal("expected force redeploy to surface tearDown error, got nil")
	}
	if !strings.Contains(err.Error(), "tear down") {
		t.Errorf("expected error to mention tear down, got: %v", err)
	}
	// State must be retained so the user can investigate / retry.
	if _, err := store.Get("handler"); err != nil {
		t.Errorf("state should be retained when teardown fails; got: %v", err)
	}
}

// TestDoUpForceTearsDownExisting verifies that force=true triggers tearDown
// of the existing function before attempting to deploy. We assert this by
// observing that the fake's Stop/Remove counters increase even though the
// new deploy will eventually fail at the health check.
func TestDoUpForceTearsDownExisting(t *testing.T) {
	tmp := setupCmdEnv(t)
	handlerPath := writeHandlerFile(t, filepath.Join(tmp, "func"))
	seedFunction(t, "handler", "stale-id", "stale-img", 5301)

	fake := &fakeRuntime{
		StartFn: func(ctx context.Context, opts runtime.StartOpts) (runtime.Container, error) {
			return runtime.Container{ID: "fake-" + opts.Name, Port: 1}, nil
		},
	}
	defer withFakeRuntime(t, fake)()

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	_ = doUp(cmd, []string{handlerPath}, true) // expected to fail at health check

	// Stop should have been called at least twice: once for the stale "stale-id"
	// (tearDown) and once for the new fake-handler (tearDownContainer on fail).
	if len(fake.StopCalls) < 2 {
		t.Errorf("expected at least 2 Stop calls (tearDown + health-fail teardown); got %v", fake.StopCalls)
	}
	foundStale := false
	for _, id := range fake.StopCalls {
		if id == "stale-id" {
			foundStale = true
			break
		}
	}
	if !foundStale {
		t.Errorf("expected stale-id in StopCalls (force tearDown); got %v", fake.StopCalls)
	}
}

// TestDoUpHealthCheckFailureTearsDownContainer is the integration regression
// test for the Phase 2 bug: when health.WaitForHealthy fails, doUp must
// stop+remove the container instead of leaving it running with stale state.
func TestDoUpHealthCheckFailureTearsDownContainer(t *testing.T) {
	tmp := setupCmdEnv(t)

	funcDir := filepath.Join(tmp, "func")
	if err := os.MkdirAll(funcDir, 0o750); err != nil {
		t.Fatal(err)
	}
	handler := []byte("package main\nfunc Handler(req map[string]any) map[string]any { return nil }\n")
	if err := os.WriteFile(filepath.Join(funcDir, "handler.go"), handler, 0o600); err != nil {
		t.Fatal(err)
	}

	fake := &fakeRuntime{
		StartFn: func(ctx context.Context, opts runtime.StartOpts) (runtime.Container, error) {
			// Port 1 is privileged + unbound; health check connect refuses immediately.
			return runtime.Container{ID: "fake-" + opts.Name, Port: 1}, nil
		},
	}
	defer withFakeRuntime(t, fake)()

	// Bound the health check via context. health.WaitForHealthy returns
	// immediately when ctx is done, so a 1.5s timeout is enough for one tick
	// (default interval is 500ms) plus teardown.
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	err := doUp(cmd, []string{filepath.Join(funcDir, "handler.go")}, false)

	if err == nil {
		t.Fatal("expected health check failure error, got nil")
	}
	if !strings.Contains(err.Error(), "Health check failed") {
		t.Errorf("expected 'Health check failed' in error, got: %v", err)
	}

	if len(fake.StopCalls) == 0 {
		t.Error("expected fake.Stop to be called on health failure; was not")
	}
	if len(fake.RemoveCalls) == 0 {
		t.Error("expected fake.Remove to be called on health failure; was not")
	}

	// Verify state was NOT recorded (a failed deploy should leave no record).
	if _, err := store.Get("handler"); err == nil {
		t.Error("expected no state record for failed deploy; got one")
	}
}
