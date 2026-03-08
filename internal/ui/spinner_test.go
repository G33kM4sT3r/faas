package ui

import (
	"errors"
	"testing"
)

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

func TestRunWithSpinnerNoColorsError(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	_, err := RunWithSpinner("testing", func() (string, error) {
		return "", errors.New("work failed")
	})
	if err == nil {
		t.Error("expected error to be propagated")
	}
	if err.Error() != "work failed" {
		t.Errorf("expected 'work failed', got %q", err.Error())
	}
}

func TestRunWithSpinnerNoColorsEmptyOutput(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	out, err := RunWithSpinner("testing", func() (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestSpinnerResultFields(t *testing.T) {
	r := SpinnerResult{Output: "hello", Err: nil}
	if r.Output != "hello" {
		t.Errorf("expected 'hello', got %q", r.Output)
	}
	if r.Err != nil {
		t.Error("expected nil error")
	}

	testErr := errors.New("test error")
	r2 := SpinnerResult{Output: "", Err: testErr}
	if !errors.Is(r2.Err, testErr) {
		t.Error("expected test error")
	}
}
