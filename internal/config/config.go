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

// ErrInvalidConfig is returned when config.toml fails validation.
var ErrInvalidConfig = errors.New("invalid config.toml")

// supportedLanguages lists the built-in language templates.
var supportedLanguages = map[string]struct{}{ //nolint:gochecknoglobals // constant lookup table
	"go":         {},
	"python":     {},
	"rust":       {},
	"php":        {},
	"typescript": {},
	"javascript": {},
}

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

// Load reads, parses, and validates a config.toml file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-supplied config path is the API contract
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate returns an error if the config is missing required fields or
// references an unsupported language.
func (c *Config) Validate() error {
	if c.Function.Name == "" {
		return fmt.Errorf("%w: function.name is required", ErrInvalidConfig)
	}
	if c.Function.Language == "" {
		return fmt.Errorf("%w: function.language is required", ErrInvalidConfig)
	}
	if c.Function.Entrypoint == "" {
		return fmt.Errorf("%w: function.entrypoint is required", ErrInvalidConfig)
	}
	if _, ok := supportedLanguages[c.Function.Language]; !ok {
		return fmt.Errorf("%w: unsupported language %q (supported: go, python, rust, php, typescript, javascript)",
			ErrInvalidConfig, c.Function.Language)
	}
	return nil
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

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
