// Package builder prepares Docker build contexts from function source and templates.
package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
	hash := hex.EncodeToString(h.Sum(nil))[:8]
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
	funcContent, err := os.ReadFile(funcPath) //nolint:gosec // user-supplied function path is the API contract
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
		Dependencies:    cfg.Dependencies.Packages,
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

	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0o600); err != nil {
		return BuildContext{}, fmt.Errorf("writing Dockerfile: %w", err)
	}

	wrapperName := wrapperOutputName(cfg.Function.Language)
	if err := os.WriteFile(filepath.Join(buildDir, wrapperName), []byte(wrapper), 0o600); err != nil {
		return BuildContext{}, fmt.Errorf("writing wrapper: %w", err)
	}

	// Go and PHP use separate handler files — user function is not embedded in template.
	// Go: handler.go (avoids duplicate package declarations and misplaced imports)
	// PHP: handler.php (avoids duplicate <?php opening tags)
	switch cfg.Function.Language {
	case "go":
		//nolint:gosec // funcContent is taint-tracked from user code; intentional copy into our temp build dir
		if err := os.WriteFile(filepath.Join(buildDir, "handler.go"), funcContent, 0o600); err != nil {
			return BuildContext{}, fmt.Errorf("writing handler.go: %w", err)
		}
	case "php":
		//nolint:gosec // funcContent is taint-tracked from user code; intentional copy into our temp build dir
		if err := os.WriteFile(filepath.Join(buildDir, "handler.php"), funcContent, 0o600); err != nil {
			return BuildContext{}, fmt.Errorf("writing handler.php: %w", err)
		}
	}

	if err := writeLanguageFiles(buildDir, cfg.Function.Language, cfg.Dependencies.Packages); err != nil {
		return BuildContext{}, fmt.Errorf("writing language files: %w", err)
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

// writeLanguageFiles generates language-specific manifest and dependency files.
// For Go/Rust: always generates manifest (go.mod/Cargo.toml) since they are required for builds.
// For Python/JS/TS/PHP: only generates when packages are non-empty.
func writeLanguageFiles(buildDir, language string, packages []string) error {
	switch language {
	case "python":
		if len(packages) == 0 {
			return nil
		}
		return writePythonRequirements(buildDir, packages)
	case "go":
		return writeGoMod(buildDir, packages)
	case "rust":
		return writeCargoToml(buildDir, packages)
	case "php":
		if len(packages) == 0 {
			return nil
		}
		return writeComposerJSON(buildDir, packages)
	case "javascript", "typescript":
		if len(packages) == 0 {
			return nil
		}
		return writeBunPackageJSON(buildDir, packages)
	}
	return nil
}

func writePythonRequirements(buildDir string, packages []string) error {
	lines := make([]string, 0, len(packages))
	for _, pkg := range packages {
		name, version := parsePackageVersion(pkg)
		if version != "" {
			lines = append(lines, name+"=="+version)
		} else {
			lines = append(lines, name)
		}
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte(content), 0o600)
}

// writeGoMod emits a minimal go.mod. Dependency require blocks are NOT
// emitted here — the Go Dockerfile runs `go get ...` + `go mod tidy` at
// build time, which both downloads the modules and updates go.mod in-place.
// See templates/go/Dockerfile.
func writeGoMod(buildDir string, _ []string) error {
	var b strings.Builder
	b.WriteString("module faas-func\n\ngo ")
	b.WriteString(goModDirective())
	b.WriteString("\n")
	return os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte(b.String()), 0o600)
}

func writeCargoToml(buildDir string, packages []string) error {
	var b strings.Builder
	b.WriteString("[package]\nname = \"faas-func\"\nversion = \"0.1.0\"\nedition = \"2024\"\n\n")
	b.WriteString("[[bin]]\nname = \"server\"\npath = \"src/main.rs\"\n\n")
	b.WriteString("[dependencies]\nserde_json = \"1\"\n")

	for _, pkg := range packages {
		name, version := parsePackageVersion(pkg)
		if version == "" {
			version = "*"
		}
		fmt.Fprintf(&b, "%s = %q\n", name, version)
	}
	return os.WriteFile(filepath.Join(buildDir, "Cargo.toml"), []byte(b.String()), 0o600)
}

func writeComposerJSON(buildDir string, packages []string) error {
	deps := make([]string, 0, len(packages))
	for _, pkg := range packages {
		name, version := parsePackageVersion(pkg)
		if version == "" {
			version = "*"
		}
		deps = append(deps, fmt.Sprintf("%q:%q", name, version))
	}
	content := fmt.Sprintf(`{"require":{%s}}`, strings.Join(deps, ","))
	return os.WriteFile(filepath.Join(buildDir, "composer.json"), []byte(content), 0o600)
}

func writeBunPackageJSON(buildDir string, packages []string) error {
	pkgJSON := fmt.Sprintf(`{"dependencies":{%s}}`, formatBunDeps(packages))
	return os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(pkgJSON), 0o600)
}

// goVersionRe matches the leading goN.M of a runtime.Version() string,
// allowing trailing patch/release/devel suffixes (go1.26.2, go1.27rc1,
// "devel go1.27-abc Tue ..."). Capture groups are major and minor.
var goVersionRe = regexp.MustCompile(`go(\d+)\.(\d+)`) //nolint:gochecknoglobals // compiled regex constant

// goModDirective returns the Go minor-version string from runtime.Version()
// suitable for a go.mod `go` directive. Wraps parseGoVersion for testability.
func goModDirective() string {
	return parseGoVersion(runtime.Version())
}

// parseGoVersion extracts the major.minor portion of a Go version string.
// Robust against release ("go1.26.2"), pre-release ("go1.27rc1"), and devel
// ("devel go1.27-abcdef ...") forms. Falls back to "1.26" on parse failure.
func parseGoVersion(v string) string {
	m := goVersionRe.FindStringSubmatch(v)
	if len(m) != 3 {
		return "1.26"
	}
	return m[1] + "." + m[2]
}

// parsePackageVersion splits "pkg@version" into (pkg, version).
// Handles scoped npm packages like "@types/node@22.0.0".
// Returns empty version string if no version specified.
func parsePackageVersion(pkg string) (name, version string) {
	if strings.HasPrefix(pkg, "@") {
		// Scoped npm: @scope/name@version — find @ after the scope
		if idx := strings.Index(pkg[1:], "@"); idx >= 0 {
			return pkg[:idx+1], pkg[idx+2:]
		}
		return pkg, ""
	}
	if idx := strings.LastIndex(pkg, "@"); idx > 0 {
		return pkg[:idx], pkg[idx+1:]
	}
	return pkg, ""
}

func formatBunDeps(packages []string) string {
	deps := make([]string, 0, len(packages))
	for _, pkg := range packages {
		name, version := parsePackageVersion(pkg)
		if version == "" {
			version = "*"
		}
		deps = append(deps, fmt.Sprintf("%q:%q", name, version))
	}
	return strings.Join(deps, ",")
}
