package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunInitGeneratesConfig(t *testing.T) {
	dir := t.TempDir()
	handler := filepath.Join(dir, "hello.py")
	if err := os.WriteFile(handler, []byte("def handler(req): return req"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := runInit(cmd, []string{handler}); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}
	cfgPath := filepath.Join(dir, "config.toml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "name = 'hello'") {
		t.Errorf("missing name: %q", body)
	}
	if !strings.Contains(body, "language = 'python'") {
		t.Errorf("missing language: %q", body)
	}
}

func TestRunInitRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runInit(cmd, []string{dir})
	if err == nil {
		t.Fatal("expected error when given a directory")
	}
	if !strings.Contains(err.Error(), "expected a function file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunInitMissingFile(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runInit(cmd, []string{"/no/such/path/handler.py"})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "cannot access") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunInitUnknownExtension(t *testing.T) {
	dir := t.TempDir()
	handler := filepath.Join(dir, "hello.cobol")
	if err := os.WriteFile(handler, []byte("01 DATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := runInit(cmd, []string{handler}); err == nil {
		t.Fatal("expected error for unknown extension")
	}
}
