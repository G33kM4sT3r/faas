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
	addUpFlags(upCmd)
}

// addUpFlags attaches the deploy-time flags to a command. Used by both
// `faas up` and `faas dev` so the dev wrapper accepts the same overrides.
func addUpFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&upPort, "port", "p", 0, "host port (0 = auto-assign)")
	cmd.Flags().StringVarP(&upName, "name", "n", "", "override function name")
	cmd.Flags().StringSliceVarP(&upEnvs, "env", "e", nil, "set env var KEY=VALUE")
	cmd.Flags().BoolVar(&upForce, "force", false, "redeploy if already running")
	cmd.Flags().BoolVar(&upNoCache, "no-cache", false, "force Docker rebuild")
}

func runUp(cmd *cobra.Command, args []string) error {
	return doUp(cmd, args, upForce)
}

func doUp(cmd *cobra.Command, args []string, force bool) error {
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
	if err := applyEnvOverrides(&cfg, upEnvs); err != nil {
		return err
	}

	if existing, err := store.Get(cfg.Function.Name); err == nil {
		if !force {
			return ui.Errorf(
				fmt.Sprintf("function %q is already running on http://localhost:%d", cfg.Function.Name, existing.Port),
				fmt.Sprintf("to redeploy: faas down %s && faas up %s", cfg.Function.Name, funcPath),
				fmt.Sprintf("to force redeploy: faas up %s --force", funcPath),
			)
		}
		// On force redeploy we must surface teardown failures: the next
		// docker run uses the same `faas-<name>` container name and would
		// otherwise fail with a confusing "name already in use" error.
		if err := tearDown(ctx, &existing); err != nil {
			return ui.Wrapf(fmt.Sprintf("failed to tear down existing %q before redeploy", cfg.Function.Name), err)
		}
	}

	if cfg.Runtime.Port > 0 {
		if err := runtime.CheckPortAvailable(ctx, cfg.Runtime.Port); err != nil {
			return err
		}
	}

	docker, err := newRuntime(ctx)
	if err != nil {
		return ui.Errorf("Cannot connect to Docker daemon", "is Docker running? Try: docker info")
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
		return ui.Wrapf("Build failed", spinErr)
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
		return ui.Wrapf("Start failed", err)
	}

	healthURL := fmt.Sprintf("http://localhost:%d%s", container.Port, cfg.Runtime.HealthPath)
	_, spinErr = ui.RunWithSpinner("Waiting for health check...", func() (string, error) {
		return "", health.WaitForHealthy(ctx, healthURL, health.Options{})
	})
	if spinErr != nil {
		logger.Warn().Str("name", cfg.Function.Name).Err(spinErr).Msg("health check failed; tearing down container")
		tearDownContainer(ctx, docker, container.ID)
		return ui.Errorf("Health check failed — container removed",
			"inspect logs above or retry: faas up "+cfg.Function.Name)
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

func resolveFuncPath(path string) (string, string, error) {
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

func applyEnvOverrides(cfg *config.Config, envs []string) error {
	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}
	for _, env := range envs {
		k, v, ok := strings.Cut(env, "=")
		if !ok {
			return ui.Errorf(fmt.Sprintf("malformed --env %q (want KEY=VALUE)", env))
		}
		if k == "" {
			return ui.Errorf(fmt.Sprintf("malformed --env %q (key is empty)", env))
		}
		cfg.Env[k] = v
	}
	return nil
}

func resolveEnvVars(env map[string]string) map[string]string {
	resolved := make(map[string]string, len(env))
	for k, v := range env {
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			envName := v[2 : len(v)-1]
			val, ok := os.LookupEnv(envName)
			if !ok {
				fmt.Printf("%s env var %q referenced in config is not set on host\n", ui.SymbolWarning, envName)
			}
			resolved[k] = val
			continue
		}
		resolved[k] = v
	}
	return resolved
}

// tearDown stops and removes a function's container during a forced
// redeploy. Unlike `faas down` (stopAndRemove), this does NOT remove the
// image — the very next deploy will use the same image tag, so wiping
// it would force an unnecessary rebuild.
func tearDown(ctx context.Context, fn *state.Function) error {
	docker, err := newRuntime(ctx)
	if err != nil {
		return fmt.Errorf("connecting to docker: %w", err)
	}

	var errs []error
	if err := docker.Stop(ctx, fn.ContainerID); err != nil {
		errs = append(errs, fmt.Errorf("stop: %w", err))
	}
	containerGone := true
	if err := docker.Remove(ctx, fn.ContainerID); err != nil {
		errs = append(errs, fmt.Errorf("remove: %w", err))
		containerGone = false
	}
	// Only wipe state when the container is actually gone — otherwise the
	// user can retry teardown against the still-existing container.
	if containerGone {
		if err := store.Remove(fn.Name); err != nil {
			errs = append(errs, fmt.Errorf("state remove: %w", err))
		}
	}
	return errors.Join(errs...)
}

// tearDownContainer stops and removes a container, logging any errors.
// Used on health-check failure paths where we want the container gone but
// the command still needs to return a clear error to the user.
func tearDownContainer(ctx context.Context, r runtime.Runtime, id string) {
	if err := r.Stop(ctx, id); err != nil {
		logger.Warn().Str("id", id).Err(err).Msg("stop failed during teardown")
	}
	if err := r.Remove(ctx, id); err != nil {
		logger.Warn().Str("id", id).Err(err).Msg("remove failed during teardown")
	}
}
