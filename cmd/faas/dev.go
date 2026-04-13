package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/ui"
)

var devCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "dev [func]",
	Short: "Run a function with hot-reload on file changes",
	Args:  cobra.ExactArgs(1),
	RunE:  runDev,
}

func setupDevFlags() {
	addUpFlags(devCmd)
}

func runDev(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	if err := doUp(cmd, args, false); err != nil {
		return err
	}

	target := args[0]
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("stat %s: %w", target, err)
	}

	fmt.Printf("%s Watching %s for changes (Ctrl+C to stop)\n", ui.SymbolInfo, target)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan error, 1)
	go func() {
		done <- watchFile(ctx, target, 250*time.Millisecond, func() {
			fmt.Printf("%s change detected — rebuilding\n", ui.SymbolInfo)
			if err := doUp(cmd, args, true); err != nil {
				fmt.Printf("%s rebuild failed: %v\n", ui.SymbolError, err)
			}
		})
	}()

	select {
	case <-sigCh:
		fmt.Printf("\n%s stopping\n", ui.SymbolInfo)
		cancel()
		return nil
	case err := <-done:
		return err
	}
}

// watchFile watches path (file or directory) and calls onChange after
// debounce has elapsed since the last fsnotify event. Returns when ctx is
// cancelled.
func watchFile(ctx context.Context, path string, debounce time.Duration, onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer func() { _ = w.Close() }()

	// Watch the parent directory when path is a file — many editors do an
	// atomic rename on save which fires create/rename events on the parent
	// rather than modify on the original inode.
	watchTarget := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		watchTarget = filepath.Dir(path)
	}
	if err := w.Add(watchTarget); err != nil {
		return fmt.Errorf("watching %s: %w", watchTarget, err)
	}

	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer == nil {
				timer = time.AfterFunc(debounce, onChange)
			} else {
				timer.Reset(debounce)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}
