// Package ui provides terminal styling and interactive components.
package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	// SymbolInfo is a blue info glyph.
	SymbolInfo = StyleURL.Render("ℹ")
)

// ColorsDisabled returns true when terminal colors should be suppressed.
func ColorsDisabled() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// Errorf formats a user-facing error with the error glyph and optional
// hint lines prefixed with an arrow.
func Errorf(msg string, hints ...string) error {
	var b strings.Builder
	b.WriteString(SymbolError)
	b.WriteString(" ")
	b.WriteString(msg)
	for _, h := range hints {
		b.WriteString("\n  → ")
		b.WriteString(h)
	}
	return errors.New(b.String())
}

// Wrapf wraps an underlying error with the error glyph and a headline.
func Wrapf(headline string, err error) error {
	return fmt.Errorf("%s %s\n  %w", SymbolError, headline, err)
}
