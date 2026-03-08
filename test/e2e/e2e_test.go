package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestLanguageLifecycle(t *testing.T) {
	if exec.Command("docker", "info").Run() != nil {
		t.Skip("docker not available")
	}

	binPath := buildBinary(t)

	tests := []struct {
		lang     string
		file     string
		expected string
	}{
		{"python", "hello.py", "Hello, Claude!"},
		{"go", "hello.go", "Hello, Claude!"},
		{"rust", "hello.rs", "Hello, Claude!"},
		{"php", "hello.php", "Hello, Claude!"},
		{"typescript", "hello.ts", "Hello, Claude!"},
		{"javascript", "hello.js", "Hello, Claude!"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			funcFile := filepath.Join("testdata", tt.file)
			name := fmt.Sprintf("e2e-%s", tt.lang)

			port := faasUp(t, binPath, funcFile, name)
			defer faasDown(t, binPath, name)

			// Verify listed
			ls := exec.Command(binPath, "ls", "--json")
			ls.Env = append(os.Environ(), "NO_COLOR=1")
			lsOut, err := ls.CombinedOutput()
			if err != nil {
				t.Fatalf("faas ls failed: %s\n%s", err, lsOut)
			}
			if !strings.Contains(string(lsOut), name) {
				t.Errorf("faas ls should list %s: %s", name, lsOut)
			}

			// Invoke function
			url := "http://localhost:" + port
			result := postJSON(t, url, `{"name":"Claude"}`)
			if result["message"] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result["message"])
			}
		})
	}
}

func TestDependencyLifecycle(t *testing.T) {
	if exec.Command("docker", "info").Run() != nil {
		t.Skip("docker not available")
	}

	binPath := buildBinary(t)

	tests := []struct {
		lang      string
		file      string
		config    string
		expected  string
		slugField string
	}{
		{"python", "hello_deps.py", "config_python.toml", "Hello, Claude!", "slug"},
		{"go", "hello_deps.go", "config_go.toml", "Hello, Claude!", "slug"},
		{"rust", "hello_deps.rs", "config_rust.toml", "Hello, Claude!", "slug"},
		{"php", "hello_deps.php", "config_php.toml", "Hello, Claude!", "slug"},
		{"javascript", "hello_deps.js", "config_javascript.toml", "Hello, Claude!", "slug"},
		{"typescript", "hello_deps.ts", "config_typescript.toml", "Hello, Claude!", "slug"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			tmpDir := t.TempDir()
			srcFunc := filepath.Join("testdata", "deps", tt.file)
			srcConfig := filepath.Join("testdata", "deps", tt.config)
			dstFunc := filepath.Join(tmpDir, tt.file)
			dstConfig := filepath.Join(tmpDir, "config.toml")

			funcData, err := os.ReadFile(srcFunc)
			if err != nil {
				t.Fatalf("reading fixture %s: %v", srcFunc, err)
			}
			if err := os.WriteFile(dstFunc, funcData, 0o644); err != nil {
				t.Fatalf("writing %s: %v", dstFunc, err)
			}
			configData, err := os.ReadFile(srcConfig)
			if err != nil {
				t.Fatalf("reading fixture %s: %v", srcConfig, err)
			}
			if err := os.WriteFile(dstConfig, configData, 0o644); err != nil {
				t.Fatalf("writing %s: %v", dstConfig, err)
			}

			name := fmt.Sprintf("e2e-deps-%s", tt.lang)

			port := faasUp(t, binPath, dstFunc, name)
			defer faasDown(t, binPath, name)

			url := "http://localhost:" + port
			result := postJSON(t, url, `{"name":"Claude"}`)
			if result["message"] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result["message"])
			}
			if result[tt.slugField] == "" {
				t.Errorf("expected non-empty %s field (proves dependency was installed)", tt.slugField)
			}
		})
	}
}

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func buildBinary(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "faas")
	rootDir := filepath.Join("..", "..")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/faas/")
	build.Dir = rootDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return binPath
}

func faasUp(t *testing.T, binPath, funcFile, name string) string {
	t.Helper()
	up := exec.Command(binPath, "up", funcFile, "--name", name)
	up.Env = append(os.Environ(), "NO_COLOR=1")
	upOut, err := up.CombinedOutput()
	if err != nil {
		t.Fatalf("faas up failed: %s\n%s", err, upOut)
	}
	t.Logf("faas up output:\n%s", stripANSI(string(upOut)))

	output := stripANSI(string(upOut))
	portIdx := strings.Index(output, "localhost:")
	if portIdx < 0 {
		t.Fatal("could not find port in output")
	}
	portStr := output[portIdx+len("localhost:"):]
	return strings.Fields(portStr)[0]
}

func faasDown(t *testing.T, binPath, name string) {
	t.Helper()
	down := exec.Command(binPath, "down", name)
	down.Env = append(os.Environ(), "NO_COLOR=1")
	downOut, err := down.CombinedOutput()
	if err != nil {
		t.Logf("faas down failed (cleanup): %s\n%s", err, downOut)
		return
	}
	output := stripANSI(string(downOut))
	if !strings.Contains(output, "Stopped and removed") {
		t.Errorf("unexpected down output: %s", output)
	}
}

func postJSON(t *testing.T, url, payload string) map[string]string {
	t.Helper()
	var (
		resp *http.Response
		err  error
	)
	for i := 0; i < 10; i++ {
		body := strings.NewReader(payload)
		resp, err = http.Post(url, "application/json", body) //nolint:noctx // test code
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("HTTP request to %s failed: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return result
}
