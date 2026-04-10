// Package logs provides structured log formatting and filtering.
package logs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/G33kM4sT3r/faas/internal/ui"
)

var levelPriority = map[string]int{ //nolint:gochecknoglobals // immutable lookup table
	"debug": 0,
	"info":  1,
	"warn":  2,
	"error": 3,
}

// logEntry is the fixed-shape decode target used by FilterByLevel.
// FormatLine decodes into a map[string]any to surface arbitrary user fields.
type logEntry struct {
	Level      string `json:"level"`
	Msg        string `json:"msg"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	DurationMS int    `json:"duration_ms"`
	Time       string `json:"time"`
}

// FormatLine parses a JSON log line and returns a styled string of the form
// "HH:MM:SS LEVEL msg [k=v ...]". Falls back to the raw line if not JSON.
// Known structural fields (level/msg/time/method/path/duration_ms) are
// rendered specially; any other fields are appended as `key=value` tails in
// sorted order for deterministic output.
func FormatLine(line string) string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return line
	}

	level, _ := raw["level"].(string)
	msg, _ := raw["msg"].(string)
	ts, _ := raw["time"].(string)
	method, _ := raw["method"].(string)
	path, _ := raw["path"].(string)
	var durationMS int
	if d, ok := raw["duration_ms"].(float64); ok {
		durationMS = int(d)
	}

	if ts == "" {
		ts = time.Now().Format("15:04:05")
	}

	display := msg
	if method != "" {
		display = method + " " + path
		if durationMS > 0 {
			display += fmt.Sprintf(" %dms", durationMS)
		}
	}

	known := map[string]struct{}{
		"level": {}, "msg": {}, "time": {},
		"method": {}, "path": {}, "duration_ms": {},
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		if _, skip := known[k]; skip {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var extras strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&extras, " %s=%v", k, raw[k])
	}

	return fmt.Sprintf("%s %s  %s%s",
		ui.StyleDim.Render(ts), formatLevel(level), display, extras.String())
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
