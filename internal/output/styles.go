package output

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Color definitions for consistent styling.
var (
	// Status colors
	ColorReady       = lipgloss.Color("42")  // Green
	ColorNotReady    = lipgloss.Color("196") // Red
	ColorProgressing = lipgloss.Color("214") // Orange/Yellow
	ColorFailed      = lipgloss.Color("196") // Red
	ColorUnknown     = lipgloss.Color("245") // Gray

	// UI colors
	ColorPrimary   = lipgloss.Color("39")  // Blue
	ColorSecondary = lipgloss.Color("245") // Gray
	ColorMuted     = lipgloss.Color("240") // Dark gray
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorWarning   = lipgloss.Color("214") // Yellow
	ColorError     = lipgloss.Color("196") // Red
)

// Styles contains lipgloss styles for output.
type Styles struct {
	// Status styles
	StatusReady       lipgloss.Style
	StatusNotReady    lipgloss.Style
	StatusProgressing lipgloss.Style
	StatusFailed      lipgloss.Style
	StatusUnknown     lipgloss.Style

	// Table styles
	TableBorder lipgloss.Style
	TableHeader lipgloss.Style
	TableCell   lipgloss.Style

	// Text styles
	Bold      lipgloss.Style
	Muted     lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style
	Highlight lipgloss.Style
}

// DefaultStyles returns the default style configuration.
func DefaultStyles() *Styles {
	return &Styles{
		// Status styles
		StatusReady:       lipgloss.NewStyle().Foreground(ColorReady).Bold(true),
		StatusNotReady:    lipgloss.NewStyle().Foreground(ColorNotReady).Bold(true),
		StatusProgressing: lipgloss.NewStyle().Foreground(ColorProgressing).Bold(true),
		StatusFailed:      lipgloss.NewStyle().Foreground(ColorFailed).Bold(true),
		StatusUnknown:     lipgloss.NewStyle().Foreground(ColorUnknown),

		// Table styles
		TableBorder: lipgloss.NewStyle().Foreground(ColorMuted),
		TableHeader: lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary),
		TableCell:   lipgloss.NewStyle(),

		// Text styles
		Bold:      lipgloss.NewStyle().Bold(true),
		Muted:     lipgloss.NewStyle().Foreground(ColorMuted),
		Success:   lipgloss.NewStyle().Foreground(ColorSuccess),
		Warning:   lipgloss.NewStyle().Foreground(ColorWarning),
		Error:     lipgloss.NewStyle().Foreground(ColorError),
		Highlight: lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
	}
}

// NoColorStyles returns styles with no colors (for NO_COLOR support).
func NoColorStyles() *Styles {
	return &Styles{
		// Status styles - no colors, just text
		StatusReady:       lipgloss.NewStyle(),
		StatusNotReady:    lipgloss.NewStyle(),
		StatusProgressing: lipgloss.NewStyle(),
		StatusFailed:      lipgloss.NewStyle(),
		StatusUnknown:     lipgloss.NewStyle(),

		// Table styles
		TableBorder: lipgloss.NewStyle(),
		TableHeader: lipgloss.NewStyle().Bold(true),
		TableCell:   lipgloss.NewStyle(),

		// Text styles
		Bold:      lipgloss.NewStyle().Bold(true),
		Muted:     lipgloss.NewStyle(),
		Success:   lipgloss.NewStyle(),
		Warning:   lipgloss.NewStyle(),
		Error:     lipgloss.NewStyle(),
		Highlight: lipgloss.NewStyle().Bold(true),
	}
}

// GetStyles returns the appropriate styles based on NO_COLOR environment variable.
func GetStyles() *Styles {
	if IsNoColor() {
		return NoColorStyles()
	}
	return DefaultStyles()
}

// IsNoColor checks if color output should be disabled.
func IsNoColor() bool {
	_, exists := os.LookupEnv("NO_COLOR")
	return exists
}

// IsTTY checks if stdout is a terminal.
func IsTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
