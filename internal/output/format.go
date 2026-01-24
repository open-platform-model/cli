// Package output provides terminal output utilities for the OPM CLI.
package output

import "strings"

// OutputFormat specifies the output format.
type OutputFormat string

const (
	// FormatYAML outputs in YAML format.
	FormatYAML OutputFormat = "yaml"

	// FormatJSON outputs in JSON format.
	FormatJSON OutputFormat = "json"

	// FormatTable outputs in table format.
	FormatTable OutputFormat = "table"

	// FormatDir outputs to a directory structure.
	FormatDir OutputFormat = "dir"
)

// String returns the string representation of the output format.
func (f OutputFormat) String() string {
	return string(f)
}

// IsValid checks if the output format is valid.
func (f OutputFormat) IsValid() bool {
	switch f {
	case FormatYAML, FormatJSON, FormatTable, FormatDir:
		return true
	default:
		return false
	}
}

// ParseOutputFormat parses a string into an OutputFormat.
// Returns FormatYAML if the string is empty or invalid.
func ParseOutputFormat(s string) OutputFormat {
	switch strings.ToLower(s) {
	case "yaml", "yml":
		return FormatYAML
	case "json":
		return FormatJSON
	case "table":
		return FormatTable
	case "dir", "directory":
		return FormatDir
	default:
		return FormatYAML
	}
}

// ValidFormats returns a slice of valid output format strings.
func ValidFormats() []string {
	return []string{"yaml", "json", "table", "dir"}
}

// ValidBuildFormats returns valid formats for build commands.
func ValidBuildFormats() []string {
	return []string{"yaml", "json", "dir"}
}

// ValidStatusFormats returns valid formats for status commands.
func ValidStatusFormats() []string {
	return []string{"table", "json", "yaml"}
}
