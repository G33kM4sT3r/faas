package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupCreatesLogDir(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	logger := Setup(logDir, true)
	logger.Info().Msg("test message")

	logFile := filepath.Join(logDir, "faas.log")
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("log file is empty")
	}
}

func TestSetupVerboseSetsDebugLevel(t *testing.T) {
	dir := t.TempDir()
	logger := Setup(dir, true)

	if !logger.Debug().Enabled() {
		t.Error("expected debug level to be enabled in verbose mode")
	}
}

func TestSetupNonVerboseSetsWarnLevel(t *testing.T) {
	dir := t.TempDir()
	logger := Setup(dir, false)

	if logger.Debug().Enabled() {
		t.Error("expected debug level to be disabled in non-verbose mode")
	}
	if !logger.Warn().Enabled() {
		t.Error("expected warn level to be enabled")
	}
}
