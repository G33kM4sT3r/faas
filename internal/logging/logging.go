// Package logging provides structured logging with file rotation.
package logging

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Setup initializes a zerolog logger writing to a rotating log file.
// verbose=true sets debug level; verbose=false sets warn level.
func Setup(logDir string, verbose bool) zerolog.Logger {
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	writer := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "faas.log"),
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}

	level := zerolog.WarnLevel
	if verbose {
		level = zerolog.DebugLevel
	}

	return zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Logger()
}
