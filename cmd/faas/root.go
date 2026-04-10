package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/logging"
	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
)

var ( //nolint:gochecknoglobals // cobra CLI state
	verbose bool
	logger  zerolog.Logger
	store   *state.Store
)

// newRuntime is the constructor for the container runtime used by commands.
// Tests replace this to inject fakes.
var newRuntime = func(ctx context.Context) (runtime.Runtime, error) { //nolint:gochecknoglobals // test hook
	return runtime.NewDocker(ctx)
}

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra root command
	Use:   "faas",
	Short: "Function as a Service — deploy functions as containers",
	Long:  "faas deploys stateless functions as containerized HTTP services.",
}

// Execute runs the root command with injected build metadata.
func Execute(version, commit string) error {
	rootCmd.Version = fmt.Sprintf("%s (%s)", version, commit)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		faasDir := faasHome()
		logger = logging.Setup(filepath.Join(faasDir, "logs"), verbose)
		store = state.New(filepath.Join(faasDir, "state.json"))
	}

	setupUpFlags()
	setupDownFlags()
	setupLsFlags()
	setupLogsFlags()
	setupInvokeFlags()

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(invokeCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(completionCmd)

	return rootCmd.Execute()
}

func faasHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".faas")
	}
	return filepath.Join(home, ".faas")
}
