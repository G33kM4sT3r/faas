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

func TestLoadMetaAllLanguages(t *testing.T) {
	engine, _ := NewEngine("")

	languages := []string{"python", "go", "rust", "php", "typescript", "javascript"}
	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			meta, err := engine.LoadMeta(lang)
			if err != nil {
				t.Fatalf("failed to load meta for %s: %v", lang, err)
			}
			if meta.Name == "" {
				t.Error("expected non-empty name")
			}
			if meta.Port == 0 {
				t.Error("expected non-zero port")
			}
			if meta.HealthPath == "" {
				t.Error("expected non-empty health path")
			}
			if meta.BaseImage == "" {
				t.Error("expected non-empty base image")
			}
			if len(meta.Extensions) == 0 {
				t.Error("expected at least one extension")
			}
		})
	}
}

func TestLoadMetaInvalidLanguage(t *testing.T) {
	engine, _ := NewEngine("")

	_, err := engine.LoadMeta("nonexistent_lang")
	if err == nil {
		t.Error("expected error for invalid language")
	}
}

func TestLoadMetaCustomOverride(t *testing.T) {
	customDir := t.TempDir()
	langDir := filepath.Join(customDir, "python")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `name = "custom-python"
extensions = [".py"]
port = 9999
health_path = "/custom"
base_image = "custom:latest"
`
	if err := os.WriteFile(filepath.Join(langDir, "template.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, _ := NewEngine(customDir)
	meta, err := engine.LoadMeta("python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "custom-python" {
		t.Errorf("expected custom-python, got %q", meta.Name)
	}
	if meta.Port != 9999 {
		t.Errorf("expected port 9999, got %d", meta.Port)
	}
}

func TestLoadMetaCustomInvalidTOML(t *testing.T) {
	customDir := t.TempDir()
	langDir := filepath.Join(customDir, "python")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "template.toml"), []byte("{bad toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, _ := NewEngine(customDir)
	_, err := engine.LoadMeta("python")
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestRenderWrapperAllLanguages(t *testing.T) {
	engine, _ := NewEngine("")
	data := &RenderData{
		UserFunction: "test_function_body",
		Port:         8080,
		HealthPath:   "/health",
		BaseImage:    "test:latest",
	}

	languages := []string{"python", "go", "rust", "php", "typescript", "javascript"}
	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			result, err := engine.RenderWrapper(lang, data)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}
			switch lang {
			case "go":
				// Go uses separate handler.go — wrapper calls Handler() but doesn't embed user function
				if !strings.Contains(result, "Handler(") {
					t.Error("Go wrapper should call Handler()")
				}
			case "php":
				// PHP uses separate handler.php — wrapper requires it
				if !strings.Contains(result, "handler.php") {
					t.Error("PHP wrapper should require handler.php")
				}
			default:
				if !strings.Contains(result, "test_function_body") {
					t.Error("rendered output should contain user function")
				}
			}
		})
	}
}

func TestRenderDockerfilePHPInstallsExtensions(t *testing.T) {
	engine, _ := NewEngine("")
	data := &RenderData{
		Port:       8080,
		HealthPath: "/health",
		BaseImage:  "php:8.5-cli-alpine3.23",
	}

	result, err := engine.RenderDockerfile("php", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(result, "docker-php-ext-install") {
		t.Error("PHP Dockerfile should install extensions via docker-php-ext-install")
	}
	// Only extensions NOT already built into php:8.5-cli-alpine3.23.
	// Built-in: mbstring, curl, dom, fileinfo, xml, PDO, pdo_sqlite, opcache, etc.
	requiredExts := []string{"intl", "zip", "bcmath", "sockets", "pcntl", "pdo_mysql", "pdo_pgsql", "gd"}
	for _, ext := range requiredExts {
		if !strings.Contains(result, ext) {
			t.Errorf("PHP Dockerfile should install extension %q", ext)
		}
	}
	if !strings.Contains(result, "docker-php-ext-configure gd") {
		t.Error("PHP Dockerfile should configure gd with freetype/jpeg before install")
	}
}

func TestRenderDockerfileAllLanguages(t *testing.T) {
	engine, _ := NewEngine("")
	data := &RenderData{
		Port:       8080,
		HealthPath: "/health",
		BaseImage:  "test:latest",
	}

	languages := []string{"python", "go", "rust", "php", "typescript", "javascript"}
	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			result, err := engine.RenderDockerfile(lang, data)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}
			if result == "" {
				t.Error("expected non-empty Dockerfile")
			}
		})
	}
}

func TestRenderWrapperInvalidLanguage(t *testing.T) {
	engine, _ := NewEngine("")
	data := &RenderData{UserFunction: "code", Port: 8080}

	_, err := engine.RenderWrapper("nonexistent_lang", data)
	if err == nil {
		t.Error("expected error for invalid language")
	}
}

func TestRenderDockerfileInvalidLanguage(t *testing.T) {
	engine, _ := NewEngine("")
	data := &RenderData{Port: 8080}

	_, err := engine.RenderDockerfile("nonexistent_lang", data)
	if err == nil {
		t.Error("expected error for invalid language")
	}
}

func TestFindWrapperFile(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"python", "server.py.tmpl"},
		{"go", "main.go.tmpl"},
		{"rust", "main.rs.tmpl"},
		{"php", "server.php.tmpl"},
		{"typescript", "server.ts.tmpl"},
		{"javascript", "server.js.tmpl"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := findWrapperFile(tt.lang)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
