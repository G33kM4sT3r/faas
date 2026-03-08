package ui

import "testing"

func TestRunWithSpinnerNoColorsRunsDirect(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	out, err := RunWithSpinner("testing", func() (string, error) {
		return "done", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "done" {
		t.Errorf("expected 'done', got %q", out)
	}
}
