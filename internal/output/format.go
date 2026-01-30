// Package output provides terminal output utilities.
package output

import "strings"

// OutputFormat specifies the output format.
type OutputFormat string

const (
	// FormatYAML outputs as YAML.
	FormatYAML OutputFormat = "yaml"

	// FormatJSON outputs as JSON.
	FormatJSON OutputFormat = "json"

	// FormatTable outputs as a formatted table.
	FormatTable OutputFormat = "table"

	// FormatDir outputs to a directory.
	FormatDir OutputFormat = "dir"
)

// Valid returns true if the format is valid.
func (f OutputFormat) Valid() bool {
	switch f {
	case FormatYAML, FormatJSON, FormatTable, FormatDir:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (f OutputFormat) String() string {
	return string(f)
}

// ParseOutputFormat parses a string into an OutputFormat.
func ParseOutputFormat(s string) (OutputFormat, bool) {
	f := OutputFormat(strings.ToLower(s))
	return f, f.Valid()
}

// ValidFormats returns all valid output format strings.
func ValidFormats() []string {
	return []string{
		string(FormatYAML),
		string(FormatJSON),
		string(FormatTable),
		string(FormatDir),
	}
}
