// Package ui provides terminal styling and interactive components.
package ui

import (
	"os"

	"charm.land/lipgloss/v2"
)

//nolint:gochecknoglobals // lipgloss styles are immutable
var (
	// StyleSuccess renders text in green for success messages.
	StyleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	// StyleError renders text in bold red for error messages.
	StyleError = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	// StyleWarning renders text in yellow for warning messages.
	StyleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	// StyleDim renders text in gray for secondary information.
	StyleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	// StyleBold renders text in bold.
	StyleBold = lipgloss.NewStyle().Bold(true)
	// StyleURL renders text as an underlined blue link.
	StyleURL = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Underline(true)

	// SymbolSuccess is a green checkmark.
	SymbolSuccess = StyleSuccess.Render("✓")
	// SymbolError is a red cross.
	SymbolError = StyleError.Render("✗")
	// SymbolWarning is a yellow exclamation mark.
	SymbolWarning = StyleWarning.Render("!")
)

// ColorsDisabled returns true when terminal colors should be suppressed.
func ColorsDisabled() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}
