package main

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/logs"
	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
	"github.com/G33kM4sT3r/faas/internal/ui"
)

var ( //nolint:gochecknoglobals // cobra flag variables
	logsFollow   bool
	logsNoFollow bool
	logsLines    int
	logsJSON     bool
	logsLevel    string
)

var logsCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "logs [func]",
	Short: "Stream function logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func setupLogsFlags() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", true, "follow log output")
	logsCmd.Flags().BoolVar(&logsNoFollow, "no-follow", false, "print and exit")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "l", 50, "number of historical lines")
	logsCmd.Flags().BoolVar(&logsJSON, "json", false, "raw JSON output")
	logsCmd.Flags().StringVar(&logsLevel, "level", "", "filter by log level")
}

func runLogs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	fn, err := store.Get(name)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return fmt.Errorf("%s Function %q not found\n  → run: faas ls", ui.SymbolError, name)
		}
		return err
	}

	docker, err := runtime.NewDocker(ctx)
	if err != nil {
		return fmt.Errorf("%s Cannot connect to Docker daemon", ui.SymbolError)
	}

	follow := logsFollow && !logsNoFollow

	reader, err := docker.Logs(ctx, fn.ContainerID, runtime.LogOpts{
		Follow: follow,
		Tail:   logsLines,
	})
	if err != nil {
		return fmt.Errorf("streaming logs: %w", err)
	}
	defer func() { _ = reader.Close() }()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if logsLevel != "" {
			filtered := logs.FilterByLevel([]string{line}, logsLevel)
			if len(filtered) == 0 {
				continue
			}
		}

		if logsJSON {
			fmt.Println(line)
		} else {
			fmt.Println(logs.FormatLine(line))
		}
	}

	return scanner.Err()
}
