package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/G33kM4sT3r/faas/internal/config"
)

func TestPrepareBuildContextPython(t *testing.T) {
	funcDir := t.TempDir()
	funcFile := filepath.Join(funcDir, "hello.py")
	if err := os.WriteFile(funcFile, []byte("def handler(request):\n    return {\"msg\": \"hello\"}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "hello",
			Language:   "python",
			Entrypoint: "hello.py",
		},
		Runtime: config.Runtime{
			Port:       8080,
			HealthPath: "/health",
		},
	}

	ctx, err := PrepareBuildContext(funcDir, cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(ctx.Dir)

	if _, err := os.Stat(filepath.Join(ctx.Dir, "Dockerfile")); err != nil {
		t.Error("Dockerfile not found in build context")
	}

	if _, err := os.Stat(filepath.Join(ctx.Dir, "server.py")); err != nil {
		t.Error("server.py not found in build context")
	}

	data, _ := os.ReadFile(filepath.Join(ctx.Dir, "server.py"))
	if len(data) == 0 {
		t.Error("server.py is empty")
	}
}

func TestPrepareBuildContextWithDeps(t *testing.T) {
	funcDir := t.TempDir()
	funcFile := filepath.Join(funcDir, "hello.py")
	os.WriteFile(funcFile, []byte("def handler(r): return {}"), 0o644)

	cfg := &config.Config{
		Function: config.Function{
			Name:       "hello",
			Language:   "python",
			Entrypoint: "hello.py",
		},
		Dependencies: config.Dependencies{
			Packages: []string{"requests", "flask"},
		},
		Runtime: config.Runtime{
			Port:       8080,
			HealthPath: "/health",
		},
	}

	ctx, err := PrepareBuildContext(funcDir, cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(ctx.Dir)

	data, err := os.ReadFile(filepath.Join(ctx.Dir, "requirements.txt"))
	if err != nil {
		t.Fatal("requirements.txt not found")
	}
	if string(data) != "requests\nflask\n" {
		t.Errorf("unexpected requirements.txt content: %q", data)
	}
}

func TestImageTag(t *testing.T) {
	tag := ImageTag("hello")
	if tag == "" {
		t.Error("expected non-empty tag")
	}
	if len(tag) < len("faas-hello:") {
		t.Errorf("tag too short: %s", tag)
	}
}
