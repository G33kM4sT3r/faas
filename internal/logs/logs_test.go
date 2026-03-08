package logs

import (
	"strings"
	"testing"
)

func TestFormatLogLineJSON(t *testing.T) {
	line := `{"level":"info","method":"POST","path":"/","duration_ms":3}`
	result := FormatLine(line)
	if result == "" {
		t.Error("expected non-empty formatted output")
	}
}

func TestFormatLogLinePlainText(t *testing.T) {
	line := "some plain text log"
	result := FormatLine(line)
	if result != line {
		t.Errorf("expected plain text passthrough, got %q", result)
	}
}

func TestFormatLogLineWithTime(t *testing.T) {
	line := `{"level":"info","msg":"hello","time":"12:30:00"}`
	result := FormatLine(line)
	if !strings.Contains(result, "12:30:00") {
		t.Error("expected timestamp in output")
	}
}

func TestFormatLogLineWithoutTime(t *testing.T) {
	line := `{"level":"info","msg":"hello"}`
	result := FormatLine(line)
	if result == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatLogLineWithDuration(t *testing.T) {
	line := `{"level":"info","method":"GET","path":"/health","duration_ms":5}`
	result := FormatLine(line)
	if !strings.Contains(result, "5ms") {
		t.Error("expected duration in output")
	}
}

func TestFormatLogLineMsgOnly(t *testing.T) {
	line := `{"level":"warn","msg":"something happened"}`
	result := FormatLine(line)
	if !strings.Contains(result, "something happened") {
		t.Error("expected message in output")
	}
}

func TestFormatLevelAllLevels(t *testing.T) {
	tests := []struct {
		level    string
		contains string
	}{
		{"debug", "DBG"},
		{"info", "INF"},
		{"warn", "WRN"},
		{"error", "ERR"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := formatLevel(tt.level)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected %q in result, got %q", tt.contains, result)
			}
		})
	}
}

func TestFilterByLevel(t *testing.T) {
	lines := []string{
		`{"level":"debug","msg":"d"}`,
		`{"level":"info","msg":"i"}`,
		`{"level":"warn","msg":"w"}`,
		`{"level":"error","msg":"e"}`,
	}

	filtered := FilterByLevel(lines, "warn")
	if len(filtered) != 2 {
		t.Errorf("expected 2 lines at warn+, got %d", len(filtered))
	}
	for _, l := range filtered {
		if strings.Contains(l, "debug") || strings.Contains(l, `"info"`) {
			t.Errorf("unexpected line in filtered output: %s", l)
		}
	}
}

func TestFilterByLevelInvalidLevel(t *testing.T) {
	lines := []string{`{"level":"info","msg":"i"}`}
	filtered := FilterByLevel(lines, "nonexistent")
	if len(filtered) != 1 {
		t.Errorf("invalid level should return all lines, got %d", len(filtered))
	}
}

func TestFilterByLevelPlainTextPassthrough(t *testing.T) {
	lines := []string{
		"plain text log",
		`{"level":"debug","msg":"d"}`,
		`{"level":"error","msg":"e"}`,
	}

	filtered := FilterByLevel(lines, "error")
	if len(filtered) != 2 {
		t.Errorf("expected 2 lines (plain text + error), got %d", len(filtered))
	}
}

func TestFilterByLevelEmpty(t *testing.T) {
	filtered := FilterByLevel([]string{}, "info")
	if len(filtered) != 0 {
		t.Errorf("expected empty result, got %d", len(filtered))
	}
}

func TestFilterByLevelDebug(t *testing.T) {
	lines := []string{
		`{"level":"debug","msg":"d"}`,
		`{"level":"info","msg":"i"}`,
		`{"level":"error","msg":"e"}`,
	}

	filtered := FilterByLevel(lines, "debug")
	if len(filtered) != 3 {
		t.Errorf("debug should include all levels, got %d", len(filtered))
	}
}
