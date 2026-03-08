package main

import (
	"os"
	"testing"
	"time"

	"github.com/G33kM4sT3r/faas/internal/config"
	"github.com/G33kM4sT3r/faas/internal/state"
)

func TestFormatAgeZero(t *testing.T) {
	result := formatAge(time.Time{})
	if result != "—" {
		t.Errorf("expected dash for zero time, got %q", result)
	}
}

func TestFormatAgeSeconds(t *testing.T) {
	result := formatAge(time.Now().Add(-30 * time.Second))
	if result != "30s ago" {
		t.Errorf("expected ~30s ago, got %q", result)
	}
}

func TestFormatAgeMinutes(t *testing.T) {
	result := formatAge(time.Now().Add(-5 * time.Minute))
	if result != "5m ago" {
		t.Errorf("expected 5m ago, got %q", result)
	}
}

func TestFormatAgeHours(t *testing.T) {
	result := formatAge(time.Now().Add(-3 * time.Hour))
	if result != "3h ago" {
		t.Errorf("expected 3h ago, got %q", result)
	}
}

func TestFormatAgeDays(t *testing.T) {
	result := formatAge(time.Now().Add(-48 * time.Hour))
	if result != "2d ago" {
		t.Errorf("expected 2d ago, got %q", result)
	}
}

func TestFormatStatusAllStatuses(t *testing.T) {
	tests := []struct {
		status state.Status
	}{
		{state.StatusHealthy},
		{state.StatusError},
		{state.StatusUnhealthy},
		{state.StatusStopped},
		{state.StatusBuilding},
		{state.StatusStarting},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := formatStatus(tt.status)
			if result == "" {
				t.Errorf("expected non-empty result for status %q", tt.status)
			}
		})
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := &config.Config{}
	applyEnvOverrides(cfg, []string{"KEY=value", "FOO=bar", "EMPTY=", "NOEQ"})

	if cfg.Env["KEY"] != "value" {
		t.Errorf("expected KEY=value, got %q", cfg.Env["KEY"])
	}
	if cfg.Env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %q", cfg.Env["FOO"])
	}
	if cfg.Env["EMPTY"] != "" {
		t.Errorf("expected EMPTY='', got %q", cfg.Env["EMPTY"])
	}
	if _, ok := cfg.Env["NOEQ"]; ok {
		t.Error("NOEQ should not be set (no = sign)")
	}
}

func TestApplyEnvOverridesNilMap(t *testing.T) {
	cfg := &config.Config{}
	applyEnvOverrides(cfg, []string{"A=1"})
	if cfg.Env["A"] != "1" {
		t.Error("should initialize nil map and set value")
	}
}

func TestResolveEnvVars(t *testing.T) {
	t.Setenv("TEST_SECRET", "s3cret")

	env := map[string]string{
		"plain":    "hello",
		"from_env": "${TEST_SECRET}",
		"missing":  "${DOES_NOT_EXIST_XYZ}",
	}

	resolved := resolveEnvVars(env)
	if resolved["plain"] != "hello" {
		t.Errorf("expected plain=hello, got %q", resolved["plain"])
	}
	if resolved["from_env"] != "s3cret" {
		t.Errorf("expected from_env=s3cret, got %q", resolved["from_env"])
	}
	if resolved["missing"] != "" {
		t.Errorf("expected missing='', got %q", resolved["missing"])
	}
}

func TestResolveEnvVarsEmpty(t *testing.T) {
	resolved := resolveEnvVars(map[string]string{})
	if len(resolved) != 0 {
		t.Errorf("expected empty map, got %d entries", len(resolved))
	}
}

func TestResolveFuncPathFile(t *testing.T) {
	dir := t.TempDir()
	funcFile := dir + "/hello.py"
	if err := os.WriteFile(funcFile, []byte("pass"), 0o644); err != nil {
		t.Fatal(err)
	}

	gotDir, gotEntry, err := resolveFuncPath(funcFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDir != dir {
		t.Errorf("expected dir %q, got %q", dir, gotDir)
	}
	if gotEntry != "hello.py" {
		t.Errorf("expected entrypoint hello.py, got %q", gotEntry)
	}
}

func TestResolveFuncPathNotFound(t *testing.T) {
	_, _, err := resolveFuncPath("/nonexistent/path/file.py")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}
