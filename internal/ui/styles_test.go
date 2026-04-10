package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestColorsDisabled(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !ColorsDisabled() {
		t.Error("expected ColorsDisabled to return true when NO_COLOR is set")
	}
}

func TestErrorfNoHints(t *testing.T) {
	err := Errorf("Cannot connect to Docker daemon")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Cannot connect to Docker daemon") {
		t.Errorf("expected headline in error, got %q", msg)
	}
	if strings.Contains(msg, "→") {
		t.Errorf("expected no hint arrows when no hints, got %q", msg)
	}
}

func TestErrorfWithHints(t *testing.T) {
	err := Errorf("Build failed", "check Docker", "or try faas logs")
	msg := err.Error()
	if !strings.Contains(msg, "Build failed") {
		t.Errorf("missing headline: %q", msg)
	}
	if !strings.Contains(msg, "check Docker") {
		t.Errorf("missing first hint: %q", msg)
	}
	if !strings.Contains(msg, "or try faas logs") {
		t.Errorf("missing second hint: %q", msg)
	}
	if strings.Count(msg, "→") != 2 {
		t.Errorf("expected 2 hint arrows, got %d in %q", strings.Count(msg, "→"), msg)
	}
}

func TestWrapfPreservesUnderlying(t *testing.T) {
	cause := errors.New("connection refused")
	wrapped := Wrapf("Build failed", cause)
	if !errors.Is(wrapped, cause) {
		t.Error("expected errors.Is to find the wrapped cause")
	}
	if !strings.Contains(wrapped.Error(), "Build failed") {
		t.Errorf("expected 'Build failed' in wrapped error: %q", wrapped.Error())
	}
	if !strings.Contains(wrapped.Error(), "connection refused") {
		t.Errorf("expected cause text in wrapped error: %q", wrapped.Error())
	}
}

func TestStylesRenderNonEmpty(t *testing.T) {
	tests := []struct {
		name  string
		style func(string) string
	}{
		{"success", func(s string) string { return StyleSuccess.Render(s) }},
		{"error", func(s string) string { return StyleError.Render(s) }},
		{"warning", func(s string) string { return StyleWarning.Render(s) }},
		{"dim", func(s string) string { return StyleDim.Render(s) }},
		{"bold", func(s string) string { return StyleBold.Render(s) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.style("test")
			if result == "" {
				t.Error("expected non-empty styled output")
			}
		})
	}
}
