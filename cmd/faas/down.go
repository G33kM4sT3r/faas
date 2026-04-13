package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
	"github.com/G33kM4sT3r/faas/internal/ui"
)

var ( //nolint:gochecknoglobals // cobra flag variables
	downAll       bool
	downKeepImage bool
)

var downCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "down [func]",
	Short: "Stop and remove a running function",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDown,
}

func setupDownFlags() {
	downCmd.Flags().BoolVar(&downAll, "all", false, "stop and remove all functions")
	downCmd.Flags().BoolVar(&downKeepImage, "keep-image", false, "don't remove the Docker image")
}

func runDown(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	docker, err := newRuntime(ctx)
	if err != nil {
		return ui.Errorf("Cannot connect to Docker daemon", "is Docker running? Try: docker info")
	}

	if downAll {
		if len(args) > 0 {
			return ui.Errorf("--all does not take a function name", "drop the argument or omit --all")
		}
		fns, err := store.List()
		if err != nil {
			return err
		}
		var errs []error
		for i := range fns {
			if err := stopAndRemove(ctx, docker, &fns[i], downKeepImage); err != nil {
				errs = append(errs, err)
				fmt.Printf("%s Failed to fully remove %q: %v\n", ui.SymbolError, fns[i].Name, err)
				continue
			}
			fmt.Printf("%s Stopped and removed %q\n", ui.SymbolSuccess, fns[i].Name)
		}
		return errors.Join(errs...)
	}

	if len(args) == 0 {
		return errors.New("provide a function name, or use --all")
	}

	name := args[0]
	fn, err := store.Get(name)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			fns, _ := store.List()
			var b strings.Builder
			b.WriteString(ui.SymbolError)
			b.WriteString(" Function ")
			b.WriteString(strconv.Quote(name))
			b.WriteString(" not found\n")
			if len(fns) > 0 {
				b.WriteString("\n  Running functions:\n")
				for i := range fns {
					fmt.Fprintf(&b, "    %s  %s  :%d  %s\n", fns[i].Name, fns[i].Language, fns[i].Port, fns[i].Status)
				}
			}
			return errors.New(b.String())
		}
		return err
	}

	if err := stopAndRemove(ctx, docker, &fn, downKeepImage); err != nil {
		return ui.Wrapf(fmt.Sprintf("Failed to fully remove %q", fn.Name), err)
	}
	fmt.Printf("%s Stopped and removed %q\n", ui.SymbolSuccess, fn.Name)
	return nil
}

// stopAndRemove runs Stop, Remove, optionally RemoveImage, and removes the
// function from state. State is wiped only when the container is gone —
// otherwise it stays so the user can retry. Cleanup errors are aggregated
// via errors.Join so every failure surfaces. Image removal is best-effort
// and never blocks state cleanup (an image may be shared across functions).
func stopAndRemove(ctx context.Context, r runtime.Runtime, fn *state.Function, keepImage bool) error {
	var errs []error
	if err := r.Stop(ctx, fn.ContainerID); err != nil {
		errs = append(errs, fmt.Errorf("stop %s: %w", fn.Name, err))
	}
	containerGone := true
	if err := r.Remove(ctx, fn.ContainerID); err != nil {
		errs = append(errs, fmt.Errorf("remove %s: %w", fn.Name, err))
		containerGone = false
	}
	if !keepImage {
		if err := r.RemoveImage(ctx, fn.ImageID); err != nil {
			logger.Debug().Err(err).Str("image", fn.ImageID).Msg("image removal failed (possibly shared)")
		}
	}
	if containerGone {
		if err := store.Remove(fn.Name); err != nil {
			errs = append(errs, fmt.Errorf("state remove %s: %w", fn.Name, err))
		}
	}
	return errors.Join(errs...)
}
