package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineRenderPython(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	userFunc := "def handler(request):\n    return {\"message\": \"hello\"}"

	data := &RenderData{
		UserFunction:    userFunc,
		Port:            8080,
		HealthPath:      "/health",
		BaseImage:       "python:3.14-alpine3.23",
		HasDependencies: false,
	}

	output, err := engine.RenderWrapper("python", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(output, "def handler(request)") {
		t.Error("rendered output should contain user function")
	}
	if !strings.Contains(output, "8080") {
		t.Error("rendered output should contain port")
	}
}

func TestEngineRenderDockerfile(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := &RenderData{
		Port:            8080,
		BaseImage:       "python:3.14-alpine3.23",
		HasDependencies: true,
	}

	output, err := engine.RenderDockerfile("python", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(output, "python:3.14-alpine3.23") {
		t.Error("Dockerfile should contain base image")
	}
	if !strings.Contains(output, "requirements.txt") {
		t.Error("Dockerfile should contain dependency step when HasDependencies=true")
	}
}

func TestEngineRenderDockerfileWithRuntimeImage(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := &RenderData{
		Port:         8080,
		BaseImage:    "golang:1.26-alpine3.23",
		RuntimeImage: "alpine:3.23",
	}

	output, err := engine.RenderDockerfile("go", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(output, "golang:1.26-alpine3.23") {
		t.Error("Dockerfile should contain base image in builder stage")
	}
	if !strings.Contains(output, "alpine:3.23") {
		t.Error("Dockerfile should contain runtime image in runtime stage")
	}
}

func TestEngineCustomTemplateOverride(t *testing.T) {
	customDir := t.TempDir()
	langDir := filepath.Join(customDir, "python")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}

	customTmpl := "CUSTOM: {{.UserFunction}}"
	tmplPath := filepath.Join(langDir, "server.py.tmpl")
	if err := os.WriteFile(tmplPath, []byte(customTmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	tomlContent := "name = \"python\"\nextensions = [\".py\"]\nport = 8080\nhealth_path = \"/health\"\nbase_image = \"python:3.14-alpine3.23\"\n"
	if err := os.WriteFile(filepath.Join(langDir, "template.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(langDir, "Dockerfile"), []byte("FROM {{.BaseImage}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := NewEngine(customDir)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := &RenderData{UserFunction: "my_func", Port: 8080, HealthPath: "/health"}
	output, err := engine.RenderWrapper("python", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(output, "CUSTOM: my_func") {
		t.Errorf("expected custom template output, got %q", output)
	}
}
