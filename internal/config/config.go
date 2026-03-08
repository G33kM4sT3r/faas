// Package config handles config.toml parsing and generation.
package config

import (
	"errors"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// ErrAlreadyExists is returned when Generate is called but config.toml already exists.
var ErrAlreadyExists = errors.New("config.toml already exists")

// Config represents the full config.toml structure.
type Config struct {
	Function     Function          `toml:"function"`
	Dependencies Dependencies      `toml:"dependencies"`
	Env          map[string]string `toml:"env"`
	Runtime      Runtime           `toml:"runtime"`
	Build        Build             `toml:"build"`
}

// Function holds function metadata.
type Function struct {
	Name       string `toml:"name"`
	Language   string `toml:"language"`
	Entrypoint string `toml:"entrypoint"`
}

// Dependencies holds language-specific dependency lists.
type Dependencies struct {
	Packages []string `toml:"packages"`
}

// Runtime holds runtime configuration.
type Runtime struct {
	Port       int    `toml:"port"`
	HealthPath string `toml:"health_path"`
}

// Build holds build-time overrides.
type Build struct {
	BaseImage    string `toml:"base_image"`
	RuntimeImage string `toml:"runtime_image"`
}

// Load reads and parses a config.toml file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Generate creates a config.toml with sensible defaults.
// Returns ErrAlreadyExists if the file already exists.
func Generate(path, name, language, entrypoint string) error {
	if _, err := os.Stat(path); err == nil {
		return ErrAlreadyExists
	}

	cfg := Config{
		Function: Function{
			Name:       name,
			Language:   language,
			Entrypoint: entrypoint,
		},
		Env: map[string]string{},
		Runtime: Runtime{
			Port:       0,
			HealthPath: "/health",
		},
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
