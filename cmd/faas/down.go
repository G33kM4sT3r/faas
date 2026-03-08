package main

import (
	"errors"
	"fmt"

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

	docker, err := runtime.NewDocker(ctx)
	if err != nil {
		return fmt.Errorf("%s Cannot connect to Docker daemon", ui.SymbolError)
	}

	if downAll {
		fns, err := store.List()
		if err != nil {
			return err
		}
		for i := range fns {
			_ = docker.Stop(ctx, fns[i].ContainerID)
			_ = docker.Remove(ctx, fns[i].ContainerID)
			if !downKeepImage {
				_ = docker.RemoveImage(ctx, fns[i].ImageID)
			}
			_ = store.Remove(fns[i].Name)
			fmt.Printf("%s Stopped and removed %q\n", ui.SymbolSuccess, fns[i].Name)
		}
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("provide a function name, or use --all")
	}

	name := args[0]
	fn, err := store.Get(name)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			fns, _ := store.List()
			msg := fmt.Sprintf("%s Function %q not found\n", ui.SymbolError, name)
			if len(fns) > 0 {
				msg += "\n  Running functions:\n"
				for i := range fns {
					msg += fmt.Sprintf("    %s  %s  :%d  %s\n", fns[i].Name, fns[i].Language, fns[i].Port, fns[i].Status)
				}
			}
			return fmt.Errorf("%s", msg)
		}
		return err
	}

	_ = docker.Stop(ctx, fn.ContainerID)
	_ = docker.Remove(ctx, fn.ContainerID)
	if !downKeepImage {
		_ = docker.RemoveImage(ctx, fn.ImageID)
	}
	_ = store.Remove(fn.Name)

	fmt.Printf("%s Stopped and removed %q\n", ui.SymbolSuccess, fn.Name)
	return nil
}
