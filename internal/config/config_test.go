package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := `
[function]
name = "hello"
language = "python"
entrypoint = "handler.py"

[dependencies]
packages = ["requests"]

[env]
DEBUG = "true"

[runtime]
port = 8080
health_path = "/health"

[build]
base_image = "python:3.12-slim"
`
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Function.Name != "hello" {
		t.Errorf("expected name 'hello', got %q", cfg.Function.Name)
	}
	if cfg.Function.Language != "python" {
		t.Errorf("expected language 'python', got %q", cfg.Function.Language)
	}
	if cfg.Function.Entrypoint != "handler.py" {
		t.Errorf("expected entrypoint 'handler.py', got %q", cfg.Function.Entrypoint)
	}
	if len(cfg.Dependencies.Packages) != 1 || cfg.Dependencies.Packages[0] != "requests" {
		t.Errorf("expected packages [requests], got %v", cfg.Dependencies.Packages)
	}
	if cfg.Env["DEBUG"] != "true" {
		t.Errorf("expected env DEBUG=true, got %q", cfg.Env["DEBUG"])
	}
	if cfg.Runtime.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Runtime.Port)
	}
	if cfg.Runtime.HealthPath != "/health" {
		t.Errorf("expected health_path '/health', got %q", cfg.Runtime.HealthPath)
	}
	if cfg.Build.BaseImage != "python:3.12-slim" {
		t.Errorf("expected base_image 'python:3.12-slim', got %q", cfg.Build.BaseImage)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestGenerateCreatesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	err := Generate(configPath, "hello", "python", "hello.py")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load generated config: %v", err)
	}
	if cfg.Function.Name != "hello" {
		t.Errorf("expected name 'hello', got %q", cfg.Function.Name)
	}
	if cfg.Function.Language != "python" {
		t.Errorf("expected language 'python', got %q", cfg.Function.Language)
	}
	if cfg.Function.Entrypoint != "hello.py" {
		t.Errorf("expected entrypoint 'hello.py', got %q", cfg.Function.Entrypoint)
	}
	if cfg.Runtime.Port != 0 {
		t.Errorf("expected port 0 (auto-assign), got %d", cfg.Runtime.Port)
	}
	if cfg.Runtime.HealthPath != "/health" {
		t.Errorf("expected default health_path '/health', got %q", cfg.Runtime.HealthPath)
	}
}

func TestGenerateDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(configPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Generate(configPath, "hello", "python", "hello.py")
	if err == nil {
		t.Error("expected error when config.toml already exists")
	}
}
