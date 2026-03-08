package ui

import "testing"

func TestColorsDisabled(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !ColorsDisabled() {
		t.Error("expected ColorsDisabled to return true when NO_COLOR is set")
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
