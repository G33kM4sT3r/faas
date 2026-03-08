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
