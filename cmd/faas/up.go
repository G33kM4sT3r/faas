package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/builder"
	"github.com/G33kM4sT3r/faas/internal/config"
	"github.com/G33kM4sT3r/faas/internal/health"
	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
	tmpl "github.com/G33kM4sT3r/faas/internal/template"
	"github.com/G33kM4sT3r/faas/internal/ui"
)

var ( //nolint:gochecknoglobals // cobra flag variables
	upPort    int
	upName    string
	upEnvs    []string
	upForce   bool
	upNoCache bool
)

var upCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "up [func]",
	Short: "Deploy a function as a containerized HTTP service",
	Args:  cobra.ExactArgs(1),
	RunE:  runUp,
}

func setupUpFlags() {
	upCmd.Flags().IntVarP(&upPort, "port", "p", 0, "host port (0 = auto-assign)")
	upCmd.Flags().StringVarP(&upName, "name", "n", "", "override function name")
	upCmd.Flags().StringSliceVarP(&upEnvs, "env", "e", nil, "set env var KEY=VALUE")
	upCmd.Flags().BoolVar(&upForce, "force", false, "redeploy if already running")
	upCmd.Flags().BoolVar(&upNoCache, "no-cache", false, "force Docker rebuild")
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	funcPath := args[0]

	funcDir, entrypoint, err := resolveFuncPath(funcPath)
	if err != nil {
		return err
	}

	lang, err := tmpl.Detect(entrypoint)
	if err != nil {
		return err
	}
	logger.Debug().Str("language", lang).Str("entrypoint", entrypoint).Msg("detected language")

	configPath := filepath.Join(funcDir, "config.toml")
	cfg, err := loadOrGenerateConfig(configPath, entrypoint, lang)
	if err != nil {
		return err
	}

	if upName != "" {
		cfg.Function.Name = upName
	}
	if upPort > 0 {
		cfg.Runtime.Port = upPort
	}
	applyEnvOverrides(&cfg, upEnvs)

	if existing, err := store.Get(cfg.Function.Name); err == nil {
		if !upForce {
			return fmt.Errorf("function %q is already running on http://localhost:%d\n  → to redeploy: faas down %s && faas up %s\n  → to force redeploy: faas up %s --force",
				cfg.Function.Name, existing.Port, cfg.Function.Name, funcPath, funcPath)
		}
		if err := tearDown(ctx, &existing); err != nil {
			logger.Warn().Err(err).Msg("failed to tear down existing function")
		}
	}

	if cfg.Runtime.Port > 0 {
		if err := runtime.CheckPortAvailable(ctx, cfg.Runtime.Port); err != nil {
			return err
		}
	}

	docker, err := runtime.NewDocker(ctx)
	if err != nil {
		return fmt.Errorf("%s Cannot connect to Docker daemon\n  → is Docker running? Try: docker info", ui.SymbolError)
	}

	buildCtx, err := builder.PrepareBuildContext(funcDir, &cfg, filepath.Join(faasHome(), "templates"))
	if err != nil {
		return fmt.Errorf("preparing build context: %w", err)
	}
	defer func() { _ = os.RemoveAll(buildCtx.Dir) }()

	_, spinErr := ui.RunWithSpinner("Building image...", func() (string, error) {
		img, err := docker.Build(ctx, runtime.BuildOpts{
			ContextDir: buildCtx.Dir,
			Tag:        buildCtx.ImageTag,
			NoCache:    upNoCache,
		})
		if err != nil {
			return "", err
		}
		logger.Info().Str("tag", img.Tag).Msg("image built")
		return img.Tag, nil
	})
	if spinErr != nil {
		return fmt.Errorf("%s Build failed\n  %w", ui.SymbolError, spinErr)
	}
	fmt.Printf("%s Built %s\n", ui.SymbolSuccess, ui.StyleBold.Render(buildCtx.ImageTag))

	envMap := resolveEnvVars(cfg.Env)
	container, err := docker.Start(ctx, runtime.StartOpts{
		ImageTag: buildCtx.ImageTag,
		Name:     "faas-" + cfg.Function.Name,
		Port:     cfg.Runtime.Port,
		Env:      envMap,
	})
	if err != nil {
		return fmt.Errorf("%s Start failed\n  %w", ui.SymbolError, err)
	}

	healthURL := fmt.Sprintf("http://localhost:%d%s", container.Port, cfg.Runtime.HealthPath)
	_, spinErr = ui.RunWithSpinner("Waiting for health check...", func() (string, error) {
		return "", health.WaitForHealthy(ctx, healthURL, health.Options{})
	})
	if spinErr != nil {
		_ = store.Set(&state.Function{
			Name:        cfg.Function.Name,
			Path:        funcDir,
			Language:    cfg.Function.Language,
			ContainerID: container.ID,
			ImageID:     buildCtx.ImageTag,
			Port:        container.Port,
			Status:      state.StatusError,
			CreatedAt:   time.Now().UTC(),
		})
		return fmt.Errorf("%s Health check failed\n  → run: faas logs %s", ui.SymbolError, cfg.Function.Name)
	}

	_ = store.Set(&state.Function{
		Name:        cfg.Function.Name,
		Path:        funcDir,
		Language:    cfg.Function.Language,
		ContainerID: container.ID,
		ImageID:     buildCtx.ImageTag,
		Port:        container.Port,
		Status:      state.StatusHealthy,
		CreatedAt:   time.Now().UTC(),
	})

	fmt.Printf("%s Running on %s\n", ui.SymbolSuccess,
		ui.StyleURL.Render(fmt.Sprintf("http://localhost:%d", container.Port)))
	fmt.Printf("%s Health check passed\n", ui.SymbolSuccess)
	return nil
}

func resolveFuncPath(path string) (dir, entrypoint string, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot access %s: %w", path, err)
	}

	if info.IsDir() {
		configPath := filepath.Join(path, "config.toml")
		cfg, err := config.Load(configPath)
		if err != nil {
			return "", "", fmt.Errorf("directory mode requires config.toml: %w", err)
		}
		return path, cfg.Function.Entrypoint, nil
	}

	return filepath.Dir(path), filepath.Base(path), nil
}

func loadOrGenerateConfig(configPath, entrypoint, lang string) (config.Config, error) {
	cfg, err := config.Load(configPath)
	if err == nil {
		return cfg, nil
	}

	name := strings.TrimSuffix(entrypoint, filepath.Ext(entrypoint))
	if genErr := config.Generate(configPath, name, lang, entrypoint); genErr != nil {
		if errors.Is(genErr, config.ErrAlreadyExists) {
			return config.Config{}, fmt.Errorf("config.toml exists but failed to parse: %w", err)
		}
		return config.Config{}, genErr
	}

	fmt.Printf("%s Generated config.toml\n", ui.SymbolSuccess)
	return config.Load(configPath)
}

func applyEnvOverrides(cfg *config.Config, envs []string) {
	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}
	for _, env := range envs {
		k, v, ok := strings.Cut(env, "=")
		if ok {
			cfg.Env[k] = v
		}
	}
}

func resolveEnvVars(env map[string]string) map[string]string {
	resolved := make(map[string]string, len(env))
	for k, v := range env {
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			envName := v[2 : len(v)-1]
			resolved[k] = os.Getenv(envName)
		} else {
			resolved[k] = v
		}
	}
	return resolved
}

func tearDown(ctx context.Context, fn *state.Function) error {
	docker, err := runtime.NewDocker(ctx)
	if err != nil {
		return err
	}
	_ = docker.Stop(ctx, fn.ContainerID)
	_ = docker.Remove(ctx, fn.ContainerID)
	return store.Remove(fn.Name)
}
