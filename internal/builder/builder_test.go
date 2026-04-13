package builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/G33kM4sT3r/faas/internal/config"
)

func TestPrepareBuildContextPython(t *testing.T) {
	funcDir := t.TempDir()
	funcFile := filepath.Join(funcDir, "hello.py")
	if err := os.WriteFile(funcFile, []byte("def handler(request):\n    return {\"msg\": \"hello\"}"), 0o600); err != nil {
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
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

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
	os.WriteFile(funcFile, []byte("def handler(r): return {}"), 0o600)

	cfg := &config.Config{
		Function: config.Function{
			Name:       "hello",
			Language:   "python",
			Entrypoint: "hello.py",
		},
		Dependencies: config.Dependencies{
			Packages: []string{"requests@2.31.0", "flask"},
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
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

	data, err := os.ReadFile(filepath.Join(ctx.Dir, "requirements.txt"))
	if err != nil {
		t.Fatal("requirements.txt not found")
	}
	if string(data) != "requests==2.31.0\nflask\n" {
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

func TestPrepareBuildContextPHPHandlerSeparateFile(t *testing.T) {
	funcDir := t.TempDir()
	// User writes a complete PHP file with <?php tag and use statements
	phpCode := "<?php\nuse Some\\Library\\Thing;\n\nfunction handler(array $input): array {\n    return ['ok' => true];\n}\n"
	if err := os.WriteFile(filepath.Join(funcDir, "hello.php"), []byte(phpCode), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "test-php-handler",
			Language:   "php",
			Entrypoint: "hello.php",
		},
		Runtime: config.Runtime{Port: 8080, HealthPath: "/health"},
	}

	ctx, err := PrepareBuildContext(funcDir, cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

	// handler.php should exist as a separate file (not embedded in server.php)
	handlerData, err := os.ReadFile(filepath.Join(ctx.Dir, "handler.php"))
	if err != nil {
		t.Fatal("handler.php not found in build context")
	}
	if !strings.Contains(string(handlerData), "<?php") {
		t.Error("handler.php should contain the original <?php tag")
	}

	// server.php should NOT contain <?php twice — it should require handler.php instead
	serverData, err := os.ReadFile(filepath.Join(ctx.Dir, "server.php"))
	if err != nil {
		t.Fatal("server.php not found")
	}
	serverContent := string(serverData)
	if strings.Contains(serverContent, "use Some\\Library\\Thing") {
		t.Error("server.php should NOT embed user function code")
	}
	if !strings.Contains(serverContent, "handler.php") {
		t.Error("server.php should require handler.php")
	}
}

func TestPrepareBuildContextGoHandlerSeparateFile(t *testing.T) {
	funcDir := t.TempDir()
	// User writes a complete Go file with package, imports, and handler
	goCode := "package main\n\nimport \"strings\"\n\nfunc Handler(req map[string]any) map[string]any {\n\treturn map[string]any{\"upper\": strings.ToUpper(\"hello\")}\n}\n"
	if err := os.WriteFile(filepath.Join(funcDir, "hello.go"), []byte(goCode), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "test-go-handler",
			Language:   "go",
			Entrypoint: "hello.go",
		},
		Runtime: config.Runtime{Port: 8080, HealthPath: "/health"},
	}

	ctx, err := PrepareBuildContext(funcDir, cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

	// handler.go should exist with original content
	handlerData, err := os.ReadFile(filepath.Join(ctx.Dir, "handler.go"))
	if err != nil {
		t.Fatal("handler.go not found in build context")
	}
	content := string(handlerData)
	if !strings.Contains(content, "package main") {
		t.Error("handler.go should contain package main")
	}
	if !strings.Contains(content, "import \"strings\"") {
		t.Error("handler.go should contain user imports")
	}

	// main.go should NOT contain user function
	mainData, err := os.ReadFile(filepath.Join(ctx.Dir, "main.go"))
	if err != nil {
		t.Fatal("main.go not found")
	}
	if strings.Contains(string(mainData), "strings.ToUpper") {
		t.Error("main.go should NOT embed user function code")
	}
}

func TestWriteLanguageFilesUnknown(t *testing.T) {
	dir := t.TempDir()
	if err := writeLanguageFiles(dir, "unknown", []string{"dep"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatBunDeps(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{"single no version", []string{"express"}, `"express":"*"`},
		{"with version", []string{"lodash@4.17.21"}, `"lodash":"4.17.21"`},
		{"multiple mixed", []string{"express", "cors@2.0"}, `"express":"*","cors":"2.0"`},
		{"empty", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBunDeps(tt.packages)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestWrapperOutputName(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"python", "server.py"},
		{"go", "main.go"},
		{"rust", "main.rs"},
		{"php", "server.php"},
		{"typescript", "server.ts"},
		{"javascript", "server.js"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := wrapperOutputName(tt.lang)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrepareBuildContextJavaScript(t *testing.T) {
	funcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(funcDir, "hello.js"), []byte("function handler(b) { return {}; }"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "hello-js",
			Language:   "javascript",
			Entrypoint: "hello.js",
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
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

	if _, err := os.Stat(filepath.Join(ctx.Dir, "server.js")); err != nil {
		t.Error("server.js not found in build context")
	}
}

func TestPrepareBuildContextJSDeps(t *testing.T) {
	funcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(funcDir, "hello.js"), []byte("function handler(b) { return {}; }"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "hello-js-deps",
			Language:   "javascript",
			Entrypoint: "hello.js",
		},
		Dependencies: config.Dependencies{
			Packages: []string{"express"},
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
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

	data, err := os.ReadFile(filepath.Join(ctx.Dir, "package.json"))
	if err != nil {
		t.Fatal("package.json not found")
	}
	if string(data) != `{"dependencies":{"express":"*"}}` {
		t.Errorf("unexpected package.json: %s", data)
	}
}

func TestPrepareBuildContextMissingFile(t *testing.T) {
	funcDir := t.TempDir()
	cfg := &config.Config{
		Function: config.Function{
			Name:       "missing",
			Language:   "python",
			Entrypoint: "nonexistent.py",
		},
		Runtime: config.Runtime{Port: 8080, HealthPath: "/health"},
	}

	_, err := PrepareBuildContext(funcDir, cfg, "")
	if err == nil {
		t.Error("expected error for missing function file")
	}
}

func TestPrepareBuildContextInvalidLanguage(t *testing.T) {
	funcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(funcDir, "hello.xyz"), []byte("code"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "invalid",
			Language:   "xyz_invalid",
			Entrypoint: "hello.xyz",
		},
		Runtime: config.Runtime{Port: 8080, HealthPath: "/health"},
	}

	_, err := PrepareBuildContext(funcDir, cfg, "")
	if err == nil {
		t.Error("expected error for invalid language")
	}
}

func TestPrepareBuildContextWithOverrides(t *testing.T) {
	funcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(funcDir, "hello.py"), []byte("def handler(r): return {}"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Function: config.Function{
			Name:       "override",
			Language:   "python",
			Entrypoint: "hello.py",
		},
		Build: config.Build{
			BaseImage:    "python:3.12-slim",
			RuntimeImage: "python:3.12-alpine",
		},
		Runtime: config.Runtime{
			Port:       9090,
			HealthPath: "/ready",
		},
	}

	ctx, err := PrepareBuildContext(funcDir, cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = os.RemoveAll(ctx.Dir) }()

	dockerfile, _ := os.ReadFile(filepath.Join(ctx.Dir, "Dockerfile"))
	content := string(dockerfile)
	if content == "" {
		t.Error("Dockerfile is empty")
	}
}

func TestWriteBunPackageJSON(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{"no versions", []string{"express", "cors"}, `{"dependencies":{"express":"*","cors":"*"}}`},
		{"with versions", []string{"lodash@4.17.21"}, `{"dependencies":{"lodash":"4.17.21"}}`},
		{"scoped", []string{"@types/node@22.0.0"}, `{"dependencies":{"@types/node":"22.0.0"}}`},
		{"scoped no version", []string{"@types/node"}, `{"dependencies":{"@types/node":"*"}}`},
		{"mixed", []string{"express", "lodash@4.17.21", "@types/node@22.0.0"}, `{"dependencies":{"express":"*","lodash":"4.17.21","@types/node":"22.0.0"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := writeBunPackageJSON(dir, tt.packages); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "package.json"))
			if err != nil {
				t.Fatal("package.json not created")
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestWritePythonRequirements(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{"no versions", []string{"flask", "requests"}, "flask\nrequests\n"},
		{"with versions", []string{"requests@2.31.0", "flask@3.0"}, "requests==2.31.0\nflask==3.0\n"},
		{"mixed", []string{"requests@2.31.0", "flask"}, "requests==2.31.0\nflask\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := writeLanguageFiles(dir, "python", tt.packages); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "requirements.txt"))
			if err != nil {
				t.Fatal("requirements.txt not created")
			}
			if string(data) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(data))
			}
		})
	}
}

// TestWriteComposerJSONIsValidJSON is the regression test for the audit fix:
// composer.json was built with fmt.Sprintf("%q:%q",...) which is *almost*
// JSON but not quite — Go's %q can emit \xNN escapes that JSON parsers
// reject. Switching to encoding/json.Marshal for value escaping ensures the
// output always parses as JSON, even with constraint specs that contain
// unusual characters (^, ~, |, spaces, etc).
func TestWriteComposerJSONIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	// Constraint specs commonly include `^`, `~`, `|`, `>=`, spaces.
	pkgs := []string{
		"vendor/pkg-a@^7.0|^8.0",
		"vendor/pkg-b@>=1.0 <2.0",
		"vendor/pkg-c@~3.4",
	}
	if err := writeComposerJSON(dir, pkgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("composer.json must be valid JSON: %v\nraw: %s", err, data)
	}
	if got := parsed["require"]["vendor/pkg-a"]; got != "^7.0|^8.0" {
		t.Errorf("vendor/pkg-a constraint: got %q, want %q", got, "^7.0|^8.0")
	}
	if got := parsed["require"]["vendor/pkg-b"]; got != ">=1.0 <2.0" {
		t.Errorf("vendor/pkg-b constraint: got %q, want %q", got, ">=1.0 <2.0")
	}
}

func TestWriteBunPackageJSONIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	pkgs := []string{
		"a-pkg@>=1.0.0 <2.0.0",
		"@scope/pkg@^7.0|^8.0",
	}
	if err := writeBunPackageJSON(dir, pkgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("package.json must be valid JSON: %v\nraw: %s", err, data)
	}
	if got := parsed["dependencies"]["@scope/pkg"]; got != "^7.0|^8.0" {
		t.Errorf("@scope/pkg constraint: got %q, want %q", got, "^7.0|^8.0")
	}
}

func TestWriteComposerJSON(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{"single", []string{"monolog/monolog@3.0"}, `{"require":{"monolog/monolog":"3.0"}}`},
		{"no version", []string{"guzzlehttp/guzzle"}, `{"require":{"guzzlehttp/guzzle":"*"}}`},
		{"multiple", []string{"monolog/monolog@3.0", "guzzlehttp/guzzle"}, `{"require":{"monolog/monolog":"3.0","guzzlehttp/guzzle":"*"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := writeComposerJSON(dir, tt.packages); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
			if err != nil {
				t.Fatal("composer.json not created")
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestWriteCargoToml(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		contains []string
	}{
		{
			"no user deps",
			nil,
			[]string{`name = "faas-func"`, `serde_json = "1"`, `path = "src/main.rs"`},
		},
		{
			"with user deps",
			[]string{"tokio@1.0", "reqwest"},
			[]string{`serde_json = "1"`, `tokio = "1.0"`, `reqwest = "*"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := writeCargoToml(dir, tt.packages); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
			if err != nil {
				t.Fatal("Cargo.toml not created")
			}
			content := string(data)
			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("Cargo.toml should contain %q, got:\n%s", s, content)
				}
			}
		})
	}
}

func TestWriteGoMod(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		contains []string
	}{
		{"no deps", nil, []string{"module faas-func", "go " + goModDirective()}},
		{"no deps empty", []string{}, []string{"module faas-func", "go " + goModDirective()}},
		{"with deps", []string{"github.com/fatih/color@v1.18.0"}, []string{"module faas-func", "go " + goModDirective()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := writeGoMod(dir, tt.packages); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err != nil {
				t.Fatal("go.mod not created")
			}
			content := string(data)
			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("go.mod should contain %q, got:\n%s", s, content)
				}
			}
		})
	}
}

func TestWriteGoModNoDepsNoRequire(t *testing.T) {
	dir := t.TempDir()
	if err := writeGoMod(dir, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	if strings.Contains(string(data), "require") {
		t.Error("go.mod without deps should not have require block")
	}
}

func TestParsePackageVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pkgName string
		version string
	}{
		{"no version", "requests", "requests", ""},
		{"with version", "requests@2.31.0", "requests", "2.31.0"},
		{"go module", "github.com/fatih/color@v1.18.0", "github.com/fatih/color", "v1.18.0"},
		{"go module no version", "github.com/fatih/color", "github.com/fatih/color", ""},
		{"scoped npm", "@types/node@22.0.0", "@types/node", "22.0.0"},
		{"scoped npm no version", "@types/node", "@types/node", ""},
		{"rust crate", "serde@1.0", "serde", "1.0"},
		{"php package", "guzzlehttp/guzzle@^7.0", "guzzlehttp/guzzle", "^7.0"},
		{"trailing @ empty version", "mylib@", "mylib", ""},
		{"empty string", "", "", ""},
		{"pseudo version", "github.com/foo/bar@v0.0.0-20260101000000-abcdef123456", "github.com/foo/bar", "v0.0.0-20260101000000-abcdef123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, version := parsePackageVersion(tt.input)
			if name != tt.pkgName {
				t.Errorf("name: expected %q, got %q", tt.pkgName, name)
			}
			if version != tt.version {
				t.Errorf("version: expected %q, got %q", tt.version, version)
			}
		})
	}
}

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"release", "go1.26.2", "1.26"},
		{"release no patch", "go1.27", "1.27"},
		{"two digit minor", "go1.100.5", "1.100"},
		{"release candidate", "go1.27rc1", "1.27"},
		{"beta", "go1.27beta1", "1.27"},
		{"devel tip", "devel go1.27-abcdef Tue Apr 1 00:00:00 2026 +0000", "1.27"},
		{"devel no prefix", "go1.28-deadbeef", "1.28"},
		{"empty", "", "1.26"},
		{"no version", "definitely not a go version", "1.26"},
		{"missing minor", "go1", "1.26"},
		{"junk", "go.bad", "1.26"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGoVersion(tt.input); got != tt.want {
				t.Errorf("parseGoVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPrepareBuildContextAllLanguages(t *testing.T) {
	tests := []struct {
		lang       string
		file       string
		wrapper    string
		manifest   string
		funcSource string
	}{
		{"go", "hello.go", "main.go", "go.mod", "package main\nfunc Handler(r map[string]any) map[string]any { return nil }"},
		{"rust", "hello.rs", "main.rs", "Cargo.toml", "fn handler(input: Value) -> Value { json!({}) }"},
		{"php", "hello.php", "server.php", "", "<?php\nfunction handler(array $input): array { return []; }"},
		{"typescript", "hello.ts", "server.ts", "", "function handler(body: unknown): unknown { return {}; }"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			funcDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(funcDir, tt.file), []byte(tt.funcSource), 0o600); err != nil {
				t.Fatal(err)
			}

			cfg := &config.Config{
				Function: config.Function{
					Name:       "test-" + tt.lang,
					Language:   tt.lang,
					Entrypoint: tt.file,
				},
				Runtime: config.Runtime{Port: 8080, HealthPath: "/health"},
			}

			ctx, err := PrepareBuildContext(funcDir, cfg, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = os.RemoveAll(ctx.Dir) }()

			if _, err := os.Stat(filepath.Join(ctx.Dir, tt.wrapper)); err != nil {
				t.Errorf("%s not found in build context", tt.wrapper)
			}
			if _, err := os.Stat(filepath.Join(ctx.Dir, "Dockerfile")); err != nil {
				t.Error("Dockerfile not found in build context")
			}
			if tt.manifest != "" {
				if _, err := os.Stat(filepath.Join(ctx.Dir, tt.manifest)); err != nil {
					t.Errorf("%s not found in build context", tt.manifest)
				}
			}
			if tt.lang == "go" {
				if _, err := os.Stat(filepath.Join(ctx.Dir, "handler.go")); err != nil {
					t.Error("handler.go not found in Go build context")
				}
			}
		})
	}
}
