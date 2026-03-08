// Package builder prepares Docker build contexts from function source and templates.
package builder

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/G33kM4sT3r/faas/internal/config"
	"github.com/G33kM4sT3r/faas/internal/template"
)

// BuildContext contains the prepared build directory and metadata.
type BuildContext struct {
	Dir      string
	ImageTag string
}

// ImageTag generates a Docker image tag for a function.
func ImageTag(name string) string {
	h := sha256.New()
	h.Write([]byte(name))
	_, _ = fmt.Fprintf(h, "%d", os.Getpid())
	hash := fmt.Sprintf("%x", h.Sum(nil))[:8]
	return fmt.Sprintf("faas-%s:%s", name, hash)
}

// PrepareBuildContext creates a temp directory with all files needed for docker build.
func PrepareBuildContext(funcDir string, cfg *config.Config, customTemplateDir string) (BuildContext, error) {
	engine, err := template.NewEngine(customTemplateDir)
	if err != nil {
		return BuildContext{}, fmt.Errorf("creating template engine: %w", err)
	}

	meta, err := engine.LoadMeta(cfg.Function.Language)
	if err != nil {
		return BuildContext{}, fmt.Errorf("loading template metadata: %w", err)
	}

	funcPath := filepath.Join(funcDir, cfg.Function.Entrypoint)
	funcContent, err := os.ReadFile(funcPath)
	if err != nil {
		return BuildContext{}, fmt.Errorf("reading function file: %w", err)
	}

	baseImage := meta.BaseImage
	if cfg.Build.BaseImage != "" {
		baseImage = cfg.Build.BaseImage
	}

	runtimeImage := meta.RuntimeImage
	if cfg.Build.RuntimeImage != "" {
		runtimeImage = cfg.Build.RuntimeImage
	}

	port := meta.Port
	if cfg.Runtime.Port > 0 {
		port = cfg.Runtime.Port
	}

	healthPath := meta.HealthPath
	if cfg.Runtime.HealthPath != "" {
		healthPath = cfg.Runtime.HealthPath
	}

	hasDeps := len(cfg.Dependencies.Packages) > 0

	data := &template.RenderData{
		UserFunction:    string(funcContent),
		Port:            port,
		HealthPath:      healthPath,
		BaseImage:       baseImage,
		RuntimeImage:    runtimeImage,
		HasDependencies: hasDeps,
	}

	wrapper, err := engine.RenderWrapper(cfg.Function.Language, data)
	if err != nil {
		return BuildContext{}, fmt.Errorf("rendering wrapper: %w", err)
	}

	dockerfile, err := engine.RenderDockerfile(cfg.Function.Language, data)
	if err != nil {
		return BuildContext{}, fmt.Errorf("rendering Dockerfile: %w", err)
	}

	buildDir, err := os.MkdirTemp("", "faas-build-*")
	if err != nil {
		return BuildContext{}, fmt.Errorf("creating build dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		return BuildContext{}, fmt.Errorf("writing Dockerfile: %w", err)
	}

	wrapperName := wrapperOutputName(cfg.Function.Language)
	if err := os.WriteFile(filepath.Join(buildDir, wrapperName), []byte(wrapper), 0o644); err != nil {
		return BuildContext{}, fmt.Errorf("writing wrapper: %w", err)
	}

	if hasDeps {
		if err := writeDependencyFile(buildDir, cfg.Function.Language, cfg.Dependencies.Packages); err != nil {
			return BuildContext{}, fmt.Errorf("writing dependencies: %w", err)
		}
	}

	return BuildContext{
		Dir:      buildDir,
		ImageTag: ImageTag(cfg.Function.Name),
	}, nil
}

func wrapperOutputName(language string) string {
	names := map[string]string{
		"python":     "server.py",
		"go":         "main.go",
		"rust":       "main.rs",
		"php":        "server.php",
		"typescript": "server.ts",
		"javascript": "server.js",
	}
	return names[language]
}

func writeDependencyFile(buildDir, language string, packages []string) error {
	switch language {
	case "python":
		content := strings.Join(packages, "\n") + "\n"
		return os.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte(content), 0o644)
	case "go":
		return nil
	case "rust":
		return nil
	case "php":
		return nil
	case "typescript", "javascript":
		pkgJSON := fmt.Sprintf(`{"dependencies":{%s}}`, formatBunDeps(packages))
		return os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(pkgJSON), 0o644)
	}
	return nil
}

func formatBunDeps(packages []string) string {
	deps := make([]string, 0, len(packages))
	for _, pkg := range packages {
		deps = append(deps, fmt.Sprintf("%q:\"*\"", pkg))
	}
	return strings.Join(deps, ",")
}
