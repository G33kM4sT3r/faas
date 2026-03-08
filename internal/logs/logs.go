// Package logs provides structured log formatting and filtering.
package logs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/G33kM4sT3r/faas/internal/ui"
)

var levelPriority = map[string]int{ //nolint:gochecknoglobals // immutable lookup table
	"debug": 0,
	"info":  1,
	"warn":  2,
	"error": 3,
}

type logEntry struct {
	Level      string `json:"level"`
	Msg        string `json:"msg"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	DurationMS int    `json:"duration_ms"`
	Time       string `json:"time"`
}

// FormatLine parses a JSON log line and returns a styled string.
// Falls back to plain text if not valid JSON.
func FormatLine(line string) string {
	var entry logEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return line
	}

	ts := entry.Time
	if ts == "" {
		ts = time.Now().Format("15:04:05")
	}

	levelStr := formatLevel(entry.Level)

	msg := entry.Msg
	if entry.Method != "" {
		msg = fmt.Sprintf("%s %s", entry.Method, entry.Path)
		if entry.DurationMS > 0 {
			msg += fmt.Sprintf(" %dms", entry.DurationMS)
		}
	}

	return fmt.Sprintf("%s %s  %s", ui.StyleDim.Render(ts), levelStr, msg)
}

// FilterByLevel returns only lines at or above the given level.
func FilterByLevel(lines []string, minLevel string) []string {
	minPriority, ok := levelPriority[minLevel]
	if !ok {
		return lines
	}

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			filtered = append(filtered, line)
			continue
		}
		if p, ok := levelPriority[entry.Level]; ok && p >= minPriority {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func formatLevel(level string) string {
	switch level {
	case "debug":
		return ui.StyleDim.Render("DBG")
	case "info":
		return ui.StyleSuccess.Render("INF")
	case "warn":
		return ui.StyleWarning.Render("WRN")
	case "error":
		return ui.StyleError.Render("ERR")
	default:
		return level
	}
}
