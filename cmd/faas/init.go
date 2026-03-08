package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/config"
	tmpl "github.com/G33kM4sT3r/faas/internal/template"
	"github.com/G33kM4sT3r/faas/internal/ui"
)

var initCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "init [func]",
	Short: "Generate config.toml for a function",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	funcPath := args[0]

	info, err := os.Stat(funcPath)
	if err != nil {
		return fmt.Errorf("cannot access %s: %w", funcPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("expected a function file, got a directory: %s", funcPath)
	}

	dir := filepath.Dir(funcPath)
	filename := filepath.Base(funcPath)

	lang, err := tmpl.Detect(filename)
	if err != nil {
		return err
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	configPath := filepath.Join(dir, "config.toml")

	if err := config.Generate(configPath, name, lang, filename); err != nil {
		return err
	}

	fmt.Printf("%s Generated config.toml for %q (%s)\n", ui.SymbolSuccess, name, lang)
	fmt.Printf("  → %s\n", ui.StyleDim.Render(configPath))
	return nil
}
