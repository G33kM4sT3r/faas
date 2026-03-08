package template

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	toml "github.com/pelletier/go-toml/v2"

	embeddedtemplates "github.com/G33kM4sT3r/faas/templates"
)

// Meta represents the metadata in template.toml.
type Meta struct {
	Name         string   `toml:"name"`
	Extensions   []string `toml:"extensions"`
	Port         int      `toml:"port"`
	HealthPath   string   `toml:"health_path"`
	BaseImage    string   `toml:"base_image"`
	RuntimeImage string   `toml:"runtime_image"`
}

// RenderData is passed to templates during rendering.
type RenderData struct {
	UserFunction    string
	Port            int
	HealthPath      string
	BaseImage       string
	RuntimeImage    string
	HasDependencies bool
}

// Engine handles template discovery and rendering.
type Engine struct {
	customDir string
}

// NewEngine creates a template engine. customDir can be "" for embedded-only.
func NewEngine(customDir string) (*Engine, error) {
	return &Engine{customDir: customDir}, nil
}

// LoadMeta loads the template.toml for a language.
func (e *Engine) LoadMeta(language string) (Meta, error) {
	if e.customDir != "" {
		metaPath := filepath.Join(e.customDir, language, "template.toml")
		if data, err := os.ReadFile(metaPath); err == nil {
			var meta Meta
			if err := toml.Unmarshal(data, &meta); err != nil {
				return Meta{}, fmt.Errorf("parsing custom template.toml: %w", err)
			}
			return meta, nil
		}
	}

	data, err := embeddedtemplates.Embedded.ReadFile(filepath.Join(language, "template.toml"))
	if err != nil {
		return Meta{}, fmt.Errorf("template not found for language %q: %w", language, err)
	}

	var meta Meta
	if err := toml.Unmarshal(data, &meta); err != nil {
		return Meta{}, fmt.Errorf("parsing embedded template.toml: %w", err)
	}
	return meta, nil
}

// RenderWrapper renders the HTTP wrapper template with the user function embedded.
func (e *Engine) RenderWrapper(language string, data *RenderData) (string, error) {
	content, err := e.readTemplateFile(language, "wrapper")
	if err != nil {
		return "", err
	}
	return renderTemplate(content, data)
}

// RenderDockerfile renders the Dockerfile template.
func (e *Engine) RenderDockerfile(language string, data *RenderData) (string, error) {
	content, err := e.readTemplateFile(language, "dockerfile")
	if err != nil {
		return "", err
	}
	return renderTemplate(content, data)
}

func (e *Engine) readTemplateFile(language, kind string) (string, error) {
	var filename string
	switch kind {
	case "dockerfile":
		filename = "Dockerfile"
	case "wrapper":
		filename = findWrapperFile(language)
	default:
		return "", fmt.Errorf("unknown template kind: %s", kind)
	}

	if e.customDir != "" {
		path := filepath.Join(e.customDir, language, filename)
		if data, err := os.ReadFile(path); err == nil {
			return string(data), nil
		}
	}

	data, err := fs.ReadFile(embeddedtemplates.Embedded, filepath.Join(language, filename))
	if err != nil {
		return "", fmt.Errorf("template file %s/%s not found: %w", language, filename, err)
	}
	return string(data), nil
}

func findWrapperFile(language string) string {
	wrappers := map[string]string{
		"python":     "server.py.tmpl",
		"go":         "main.go.tmpl",
		"rust":       "main.rs.tmpl",
		"php":        "server.php.tmpl",
		"typescript": "server.ts.tmpl",
		"javascript": "server.js.tmpl",
	}
	if f, ok := wrappers[language]; ok {
		return f
	}
	return ""
}

func renderTemplate(tmplContent string, data *RenderData) (string, error) {
	tmpl, err := template.New("tmpl").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
